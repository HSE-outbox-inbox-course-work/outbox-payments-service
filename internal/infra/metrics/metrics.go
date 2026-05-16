package metrics

import "github.com/prometheus/client_golang/prometheus"

// Metrics содержит все Prometheus-метрики сервиса.
// Сейчас здесь только HTTP RED (Rate / Errors / Duration).
// Бизнес-метрики (transfer outcomes, pool stats, outbox depth) добавим позже.
type Metrics struct {
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
}

// New создаёт метрики и регистрирует их в переданном Registerer.
// Namespace "outbox" даёт имена вида outbox_http_requests_total.
func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "outbox",
				Name:      "http_requests_total",
				Help:      "Total HTTP requests partitioned by method, route and status code.",
			},
			[]string{"method", "route", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "outbox",
				Name:      "http_request_duration_seconds",
				Help:      "HTTP request latency in seconds.",
				Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5},
			},
			[]string{"method", "route"},
		),
	}
	reg.MustRegister(m.HTTPRequestsTotal, m.HTTPRequestDuration)
	return m
}
