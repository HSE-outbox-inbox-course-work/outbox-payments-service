package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"outbox-payment-service/internal/infra/metrics"
)

// MetricsMiddleware записывает RED-метрики для каждого HTTP-запроса:
//   - outbox_http_requests_total{method, route, status}
//   - outbox_http_request_duration_seconds{method, route}
//
// route берётся из шаблона Echo (например /api/v1/accounts/transfer-money),
// а не из реального URL — это защищает от взрыва кардинальности.
func MetricsMiddleware(m *metrics.Metrics) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			status := c.Response().Status
			if err != nil {
				var code int
				if he, ok := err.(*echo.HTTPError); ok {
					code = he.Code
				} else {
					code = http.StatusInternalServerError
				}
				status = code
			}

			route := c.Path()
			if route == "" {
				route = "unknown"
			}

			m.HTTPRequestsTotal.WithLabelValues(
				c.Request().Method,
				route,
				strconv.Itoa(status),
			).Inc()

			m.HTTPRequestDuration.WithLabelValues(
				c.Request().Method,
				route,
			).Observe(time.Since(start).Seconds())

			return err
		}
	}
}
