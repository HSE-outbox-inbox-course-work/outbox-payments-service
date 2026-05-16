// Package metrics держит описание всех Prometheus-метрик outbox-сервиса
// и их регистрацию в реестре. Сами места инкрементов разнесены по слоям:
//
//   - HTTP RED — в middleware (internal/infra/http/server/middleware);
//   - бизнес-исход перевода — в usecase MoneyTransfer;
//   - вставка события в outbox — в репозитории accounts;
//   - состояние пула pgx и глубина outbox-таблицы — фоновый коллектор Run().
//
// Namespace "outbox" даёт префикс outbox_* у всех имён, что отделяет
// сервисные метрики от метрик инфраструктуры (pg_*, kafka_*).
package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "outbox"

// Buckets для бизнес-длительностей (секунды). Перекрывают типичный диапазон
// от долей миллисекунды до пары секунд — этого достаточно для DB-операций
// и одного REST-запроса.
var latencyBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}

// Outcome — конечный набор значений метки outcome. Только перечислимые
// значения, иначе rate(...) by (outcome) даст случайные имена.
type Outcome string

const (
	OutcomeOK                Outcome = "ok"
	OutcomeInsufficientFunds Outcome = "insufficient_funds"
	OutcomeInvalidAmount     Outcome = "invalid_amount"
	OutcomeAccountNotFound   Outcome = "account_not_found"
	OutcomeDBError           Outcome = "db_error"
)

// Metrics — контейнер всех коллекторов. Передаётся по указателю в те места,
// которым нужно делать наблюдения; nil-безопасных методов нет, инстанс
// должен быть создан через New().
type Metrics struct {
	// HTTP RED
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	// Бизнес-метрики usecase
	TransferAttempts *prometheus.CounterVec
	TransferDuration *prometheus.HistogramVec

	// Outbox: запись события
	OutboxEventsInserted *prometheus.CounterVec

	// Outbox: состояние таблицы (фоновый сбор)
	OutboxTableRows         prometheus.Gauge
	OutboxOldestEventAgeSec prometheus.Gauge

	// pgx pool stats (фоновый сбор)
	PoolAcquired    prometheus.Gauge
	PoolIdle        prometheus.Gauge
	PoolTotal       prometheus.Gauge
	PoolMax         prometheus.Gauge
	PoolAcquireWait prometheus.Gauge
}

func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests partitioned by method, route and status code.",
		}, []string{"method", "route", "status"}),

		HTTPRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency in seconds, observed in middleware.",
			Buckets:   latencyBuckets,
		}, []string{"method", "route"}),

		TransferAttempts: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "transfer_attempts_total",
			Help:      "Money transfer attempts grouped by business outcome.",
		}, []string{"outcome"}),

		TransferDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "transfer_duration_seconds",
			Help:      "Duration of TransferMoney use case (the whole DB transaction), in seconds.",
			Buckets:   latencyBuckets,
		}, []string{"outcome"}),

		OutboxEventsInserted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "events_inserted_total",
			Help:      "Events inserted into the outbox table (Debezium will pick them up later).",
		}, []string{"event_type"}),

		OutboxTableRows: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "table_rows",
			Help:      "Current number of rows in the outbox table.",
		}),
		OutboxOldestEventAgeSec: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "oldest_event_age_seconds",
			Help:      "Age (seconds) of the oldest row in the outbox table; grows if CDC stops shipping.",
		}),

		PoolAcquired: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace, Subsystem: "pgx_pool",
			Name: "acquired_connections", Help: "pgxpool: currently acquired connections.",
		}),
		PoolIdle: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace, Subsystem: "pgx_pool",
			Name: "idle_connections", Help: "pgxpool: idle connections in the pool.",
		}),
		PoolTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace, Subsystem: "pgx_pool",
			Name: "total_connections", Help: "pgxpool: total connections currently in the pool.",
		}),
		PoolMax: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace, Subsystem: "pgx_pool",
			Name: "max_connections", Help: "pgxpool: configured maximum pool size.",
		}),
		PoolAcquireWait: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace, Subsystem: "pgx_pool",
			Name: "wait_count", Help: "pgxpool: cumulative number of Acquire calls that had to wait.",
		}),
	}

	reg.MustRegister(
		m.HTTPRequestsTotal, m.HTTPRequestDuration,
		m.TransferAttempts, m.TransferDuration,
		m.OutboxEventsInserted,
		m.OutboxTableRows, m.OutboxOldestEventAgeSec,
		m.PoolAcquired, m.PoolIdle, m.PoolTotal, m.PoolMax, m.PoolAcquireWait,
	)
	return m
}

// Run запускает фоновый сборщик, который раз в interval опрашивает
// состояние пула pgx и outbox-таблицы. Использует одно read-only
// соединение из пула, нагрузку держит минимальной.
//
// Останавливается по ctx.Done().
func (m *Metrics) Run(ctx context.Context, pool *pgxpool.Pool, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	m.collectOnce(ctx, pool)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.collectOnce(ctx, pool)
		}
	}
}

func (m *Metrics) collectOnce(ctx context.Context, pool *pgxpool.Pool) {
	stat := pool.Stat()
	m.PoolAcquired.Set(float64(stat.AcquiredConns()))
	m.PoolIdle.Set(float64(stat.IdleConns()))
	m.PoolTotal.Set(float64(stat.TotalConns()))
	m.PoolMax.Set(float64(stat.MaxConns()))
	m.PoolAcquireWait.Set(float64(stat.EmptyAcquireCount()))

	// Глубина и возраст таблицы outbox. Запрос лёгкий: COUNT(*) по небольшой
	// таблице, и max(created_at) использует естественный порядок вставки.
	// Если запрос упал (например, БД недоступна) — просто пропускаем тик.
	var rows int64
	var ageSec float64
	row := pool.QueryRow(ctx,
		`SELECT COUNT(*),
		        COALESCE(EXTRACT(EPOCH FROM (now() - MIN(created_at))), 0)
		 FROM outbox`)
	if err := row.Scan(&rows, &ageSec); err != nil {
		slog.Debug("metrics: cannot collect outbox stats", slog.String("err", err.Error()))
		return
	}
	m.OutboxTableRows.Set(float64(rows))
	m.OutboxOldestEventAgeSec.Set(ageSec)
}
