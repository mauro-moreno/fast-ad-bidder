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
	"github.com/fast-ad-bidder/bidder/src/services/builder"
	"github.com/fast-ad-bidder/bidder/src/services/matcher"
	"github.com/fast-ad-bidder/bidder/src/services/store"
	"github.com/fast-ad-bidder/bidder/src/services/tracker"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

func main() {
	// Load .env file if it exists (ignore error if not found - use OS env vars)
	_ = godotenv.Load()

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

	// Initialize PostgreSQL connection (T078)
	dbDSN := getEnv("DATABASE_URL", "postgres://bidder:bidder@localhost:5432/bidder?sslmode=disable")
	db, err := store.ConnectPostgres(dbDSN)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer db.Close()
	logger.Info("Connected to PostgreSQL database")

	// Initialize campaign store with PostgreSQL loader (T078)
	postgresLoader := store.NewPostgresLoader(db)
	campaignStore := store.NewMemoryCampaignStore(postgresLoader)

	// Load campaigns on startup (T078)
	ctx := context.Background()
	if err := campaignStore.LoadCampaigns(ctx); err != nil {
		logger.Fatal("Failed to load campaigns from database", zap.Error(err))
	}
	logger.Info("Initial campaign data loaded successfully")

	// Start campaign reload goroutine (T079)
	reloadInterval := 60 * time.Second
	go func() {
		ticker := time.NewTicker(reloadInterval)
		defer ticker.Stop()

		for range ticker.C {
			logger.Debug("Reloading campaigns from database")
			if err := campaignStore.RefreshCampaigns(context.Background()); err != nil {
				logger.Error("Failed to reload campaigns", zap.Error(err))
			} else {
				logger.Debug("Campaigns reloaded successfully")
			}
		}
	}()
	logger.Info("Campaign reload goroutine started", zap.Duration("interval", reloadInterval))

	// Initialize win notification tracking components (T096)
	bidCacheTTL := 5 * time.Minute
	bidCache := tracker.NewBidCache(bidCacheTTL)
	logger.Info("Bid cache initialized", zap.Duration("ttl", bidCacheTTL))

	budgetTracker := tracker.NewBudgetTracker(campaignStore, logger)
	metricsAgg := tracker.NewMetricsAggregator()
	parser := tracker.NewParser(logger)

	// Initialize InfluxDB writer (optional)
	var influxWriter *tracker.InfluxWriter
	influxURL := getEnv("INFLUXDB_URL", "")
	if influxURL != "" {
		influxToken := getEnv("INFLUXDB_TOKEN", "")
		influxOrg := getEnv("INFLUXDB_ORG", "bidder")
		influxBucket := getEnv("INFLUXDB_BUCKET", "metrics")

		influxConfig := lib.InfluxConfig{
			URL:    influxURL,
			Token:  influxToken,
			Org:    influxOrg,
			Bucket: influxBucket,
		}
		influxClient := lib.NewInfluxClient(influxConfig)
		influxWriter = tracker.NewInfluxWriter(influxClient, logger)
		logger.Info("InfluxDB writer initialized",
			zap.String("url", influxURL),
			zap.String("org", influxOrg),
			zap.String("bucket", influxBucket),
		)
	}

	// Initialize win processor with worker pool (T098)
	winQueueSize := 10000
	winWorkerCount := 4
	winProcessor := tracker.NewWinProcessor(
		bidCache,
		budgetTracker,
		metricsAgg,
		influxWriter,
		logger,
		winQueueSize,
		winWorkerCount,
	)
	winProcessor.Start(ctx)
	logger.Info("Win processor started",
		zap.Int("queue_size", winQueueSize),
		zap.Int("worker_count", winWorkerCount),
	)

	// Start budget reset scheduler (T102)
	budgetTracker.StartBudgetResetScheduler(ctx)
	logger.Info("Budget reset scheduler started")

	// Initialize bid generation services (T078)
	campaignMatcher := matcher.NewCampaignMatcher(campaignStore, logger)
	winNotificationURL := getEnv("WIN_NOTIFICATION_URL", "https://bidder.example.com/win")
	responseBuilder := builder.NewResponseBuilder(winNotificationURL)

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

	// Bid endpoint - OpenRTB 2.5 bid request handler (User Story 1 & 2, T097)
	bidHandler := api.NewBidHandler(campaignMatcher, responseBuilder, bidCache, metricsAgg, influxWriter, logger, metrics)
	e.POST("/bid", bidHandler.HandleBid)

	// Win notification endpoint - User Story 3 (T096)
	winHandler := api.NewWinHandler(parser, winProcessor, bidCache, logger)
	e.GET("/win", winHandler.HandleWin)

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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	// Close win processor and flush pending writes
	logger.Info("Flushing pending win notifications...")
	if err := winProcessor.Close(shutdownCtx); err != nil {
		logger.Error("Failed to flush win processor", zap.Error(err))
	}

	// Close InfluxDB writer if initialized
	if influxWriter != nil {
		logger.Info("Closing InfluxDB writer...")
		if err := influxWriter.Close(); err != nil {
			logger.Error("Failed to close InfluxDB writer", zap.Error(err))
		}
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
