// Package metrics регистрирует Prometheus-метрики outbox-сервиса.
// Инкременты разнесены по слоям: HTTP middleware, usecase MoneyTransfer,
// repository accounts. Фоновый коллектор Run() обновляет gauge'и pgxpool.
//
// Namespace outbox_ отделяет сервисные метрики от инфраструктурных (pg_*, kafka_*).
package metrics

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "outbox"

var latencyBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}

// Outcome — конечный набор значений метки outcome, чтобы кардинальность
// не разрасталась случайными строками.
type Outcome string

const (
	OutcomeOK                Outcome = "ok"
	OutcomeInsufficientFunds Outcome = "insufficient_funds"
	OutcomeInvalidAmount     Outcome = "invalid_amount"
	OutcomeAccountNotFound   Outcome = "account_not_found"
	OutcomeDBError           Outcome = "db_error"
)

type Metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec

	TransferAttempts *prometheus.CounterVec
	TransferDuration *prometheus.HistogramVec

	OutboxEventsInserted *prometheus.CounterVec

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
			Help:      "Events inserted into the outbox table.",
		}, []string{"event_type"}),

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
		m.PoolAcquired, m.PoolIdle, m.PoolTotal, m.PoolMax, m.PoolAcquireWait,
	)
	return m
}

// Run периодически снимает pgxpool.Stat() в gauge'и. Останавливается по ctx.Done().
func (m *Metrics) Run(ctx context.Context, pool *pgxpool.Pool, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	m.collectOnce(pool)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.collectOnce(pool)
		}
	}
}

func (m *Metrics) collectOnce(pool *pgxpool.Pool) {
	stat := pool.Stat()
	m.PoolAcquired.Set(float64(stat.AcquiredConns()))
	m.PoolIdle.Set(float64(stat.IdleConns()))
	m.PoolTotal.Set(float64(stat.TotalConns()))
	m.PoolMax.Set(float64(stat.MaxConns()))
	m.PoolAcquireWait.Set(float64(stat.EmptyAcquireCount()))
}
