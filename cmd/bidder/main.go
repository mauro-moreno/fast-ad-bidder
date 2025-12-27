package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fast-ad-bidder/bidder/src/api"
	"github.com/fast-ad-bidder/bidder/src/lib"
	"github.com/fast-ad-bidder/bidder/src/middleware"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logLevel := getEnv("LOG_LEVEL", "info")
	logger, err := lib.InitLogger(logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize metrics
	namespace := getEnv("METRICS_NAMESPACE", "bidder")
	metrics := lib.InitMetrics(namespace)

	logger.Info("Starting Fast Ad Bidder",
		zap.String("version", "0.1.0"),
		zap.String("log_level", logLevel),
	)

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Configure middleware stack
	e.Use(echomiddleware.Recover())
	e.Use(middleware.LoggingMiddleware(logger))
	e.Use(middleware.MetricsMiddleware(metrics))

	// Configure mTLS if enabled
	mtlsEnabled := getEnv("MTLS_ENABLED", "false") == "true"
	if mtlsEnabled {
		e.Use(middleware.MTLSMiddleware())
		logger.Info("mTLS authentication enabled")
	}

	// Health check endpoint (no auth required)
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"status": "healthy",
		})
	})

	// Metrics endpoint (no auth required for Prometheus scraping)
	e.GET("/metrics", api.HandleMetrics)

	// Bid endpoint - OpenRTB 2.5 bid request handler (User Story 1)
	bidHandler := api.NewBidHandler(nil, nil, logger, metrics)
	e.POST("/bid", bidHandler.HandleBid)

	// Determine server port
	port := getEnv("PORT", "8080")
	addr := fmt.Sprintf(":%s", port)

	// Start server with graceful shutdown
	go func() {
		logger.Info("Starting HTTP server", zap.String("addr", addr))

		if mtlsEnabled {
			// Load TLS config
			tlsConfig, err := middleware.CreateTLSConfig(middleware.MTLSConfig{
				CertFile: getEnv("TLS_CERT_FILE", "config/certs/server.crt"),
				KeyFile:  getEnv("TLS_KEY_FILE", "config/certs/server.key"),
				CAFile:   getEnv("TLS_CA_FILE", "config/certs/ca.crt"),
				Enabled:  true,
			})
			if err != nil {
				logger.Fatal("Failed to create TLS config", zap.Error(err))
			}

			e.TLSServer.TLSConfig = tlsConfig
			certFile := getEnv("TLS_CERT_FILE", "config/certs/server.crt")
			keyFile := getEnv("TLS_KEY_FILE", "config/certs/server.key")

			if err := e.StartTLS(addr, certFile, keyFile); err != nil {
				logger.Fatal("Failed to start TLS server", zap.Error(err))
			}
		} else {
			if err := e.Start(addr); err != nil {
				logger.Info("Server shutdown", zap.Error(err))
			}
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
