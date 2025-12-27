package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status       string            `json:"status"`
	Version      string            `json:"version"`
	Timestamp    string            `json:"timestamp"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

// HealthChecker holds dependencies for health checks
type HealthChecker struct {
	DB      *sql.DB
	Version string
}

// NewHealthChecker creates a new health checker instance
func NewHealthChecker(db *sql.DB, version string) *HealthChecker {
	return &HealthChecker{
		DB:      db,
		Version: version,
	}
}

// HandleHealth returns the health check handler
func (h *HealthChecker) HandleHealth(c echo.Context) error {
	response := HealthResponse{
		Status:       "healthy",
		Version:      h.Version,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Dependencies: make(map[string]string),
	}

	// Check database connection if available
	if h.DB != nil {
		if err := h.DB.Ping(); err != nil {
			response.Status = "unhealthy"
			response.Dependencies["postgres"] = "down"
			return c.JSON(http.StatusServiceUnavailable, response)
		}
		response.Dependencies["postgres"] = "up"
	}

	return c.JSON(http.StatusOK, response)
}

// HandleReadiness returns the readiness check handler
// Readiness checks if the service is ready to accept traffic
func (h *HealthChecker) HandleReadiness(c echo.Context) error {
	// For now, readiness is the same as health
	// In the future, this could check if campaigns are loaded, etc.
	return h.HandleHealth(c)
}

// HandleLiveness returns the liveness check handler
// Liveness checks if the service is alive (basic check)
func (h *HealthChecker) HandleLiveness(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "alive",
	})
}
