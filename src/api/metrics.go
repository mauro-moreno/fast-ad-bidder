package api

import (
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HandleMetrics returns the Prometheus metrics handler
// This endpoint exposes metrics in Prometheus text format for scraping
func HandleMetrics(c echo.Context) error {
	// Use the Prometheus HTTP handler to serve metrics
	handler := promhttp.Handler()
	handler.ServeHTTP(c.Response(), c.Request())
	return nil
}
