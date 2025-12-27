package middleware

import (
	"time"

	"github.com/fast-ad-bidder/bidder/src/lib"
	"github.com/labstack/echo/v4"
)

// MetricsMiddleware tracks request metrics for Prometheus
func MetricsMiddleware(metrics *lib.Metrics) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Process request
			err := next(c)

			// Record latency
			duration := time.Since(start).Seconds()
			metrics.LatencyHistogram.WithLabelValues(c.Path()).Observe(duration)

			// Increment request counter based on endpoint
			if c.Path() == "/bid" {
				if err == nil && c.Response().Status < 400 {
					metrics.RequestsTotal.WithLabelValues("valid").Inc()
				} else {
					metrics.RequestsTotal.WithLabelValues("invalid").Inc()
				}
			}

			return err
		}
	}
}
