package middleware

import (
	"time"

	"github.com/fast-ad-bidder/bidder/src/lib"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// LoggingMiddleware logs HTTP requests with correlation IDs
func LoggingMiddleware(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Generate correlation ID
			correlationID := lib.GenerateCorrelationID()
			c.Set(lib.CorrelationIDKey, correlationID)

			// Add correlation ID to response header
			c.Response().Header().Set("X-Correlation-ID", correlationID)

			// Process request
			err := next(c)

			// Log request details
			req := c.Request()
			res := c.Response()
			latency := time.Since(start)

			fields := []zap.Field{
				zap.String("correlation_id", correlationID),
				zap.String("method", req.Method),
				zap.String("path", req.URL.Path),
				zap.Int("status", res.Status),
				zap.Duration("latency", latency),
				zap.String("remote_ip", c.RealIP()),
				zap.String("user_agent", req.UserAgent()),
			}

			if err != nil {
				fields = append(fields, zap.Error(err))
				logger.Error("Request failed", fields...)
			} else {
				logger.Info("Request completed", fields...)
			}

			return err
		}
	}
}
