package api

import (
	"net/http"

	"github.com/fast-ad-bidder/bidder/src/lib"
	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/tracker"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// WinHandler handles win notification requests
// T095: Implement win notification endpoint handler
type WinHandler struct {
	parser       *tracker.Parser
	winProcessor *tracker.WinProcessor
	bidCache     *tracker.BidCache
	logger       *zap.Logger
	winsTotal    prometheus.Counter
}

var (
	// T100: Add Prometheus counters for wins received
	winsReceivedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wins_received_total",
		Help: "Total number of win notifications received",
	})

	winsProcessedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wins_processed_total",
		Help: "Total number of win notifications successfully processed",
	})

	winsOrphanedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wins_orphaned_total",
		Help: "Total number of orphaned win notifications (bid not found in cache)",
	})

	winsDroppedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "wins_dropped_total",
		Help: "Total number of win notifications dropped due to queue overflow",
	})
)

// NewWinHandler creates a new win notification handler
func NewWinHandler(
	parser *tracker.Parser,
	winProcessor *tracker.WinProcessor,
	bidCache *tracker.BidCache,
	logger *zap.Logger,
) *WinHandler {
	return &WinHandler{
		parser:       parser,
		winProcessor: winProcessor,
		bidCache:     bidCache,
		logger:       logger,
		winsTotal:    winsReceivedTotal,
	}
}

// HandleWin processes win notification requests
// GET /win?bid=<bid_id>&price=<price>&currency=<currency>
func (wh *WinHandler) HandleWin(c echo.Context) error {
	// Get correlation ID from context (set by logging middleware)
	correlationID, _ := c.Get(lib.CorrelationIDKey).(string)

	// Increment total wins received counter
	winsReceivedTotal.Inc()

	wh.logger.Info("Win notification request received",
		zap.String("correlation_id", correlationID),
		zap.String("query", c.Request().URL.RawQuery),
	)

	// Parse win notification from query parameters
	winNotif, errResp := wh.parser.ParseWinNotification(c.Request().URL.Query(), correlationID)
	if errResp != nil {
		wh.logger.Warn("Invalid win notification",
			zap.String("correlation_id", correlationID),
			zap.String("error", errResp.Error),
		)
		return c.JSON(http.StatusBadRequest, errResp)
	}

	// Look up bid in cache to validate it exists
	cachedBid, found := wh.bidCache.Get(winNotif.BidID)
	if !found {
		// Orphaned win - bid not found or expired
		winsOrphanedTotal.Inc()

		wh.logger.Warn("Win notification for unknown or expired bid",
			zap.String("correlation_id", correlationID),
			zap.String("bid_id", winNotif.BidID),
			zap.Float64("price", winNotif.Price),
		)

		errResp := models.NewErrorResponse(
			"Bid not found or expired",
			models.ErrorCodeBidNotFound,
		).WithCorrelationID(correlationID)

		return c.JSON(http.StatusNotFound, errResp)
	}

	// Queue win notification for async processing
	wh.winProcessor.QueueWin(winNotif)

	// Increment processed counter (queued successfully)
	winsProcessedTotal.Inc()

	wh.logger.Info("Win notification queued for processing",
		zap.String("correlation_id", correlationID),
		zap.String("bid_id", winNotif.BidID),
		zap.String("campaign_id", cachedBid.CampaignID),
		zap.Float64("price", winNotif.Price),
	)

	// Return 200 OK with minimal response (pixel tracking)
	return c.NoContent(http.StatusOK)
}
