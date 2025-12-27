package api

import (
	"net/http"

	"github.com/fast-ad-bidder/bidder/src/lib"
	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/matcher"
	"github.com/fast-ad-bidder/bidder/src/services/validator"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// BidHandler handles OpenRTB bid requests
type BidHandler struct {
	parser    *validator.Parser
	validator *validator.Validator
	matcher   matcher.Matcher // Will be used in User Story 2
	logger    *zap.Logger
	metrics   *lib.Metrics
}

// NewBidHandler creates a new bid handler
func NewBidHandler(matcher matcher.Matcher, _ interface{}, logger *zap.Logger, metrics *lib.Metrics) *BidHandler {
	return &BidHandler{
		parser:    validator.NewParser(logger),
		validator: validator.NewValidator(logger, metrics),
		matcher:   matcher,
		logger:    logger,
		metrics:   metrics,
	}
}

// HandleBid processes incoming OpenRTB bid requests
func (h *BidHandler) HandleBid(c echo.Context) error {
	// Get correlation ID from context (set by logging middleware)
	correlationID, _ := c.Get(lib.CorrelationIDKey).(string)

	// Parse bid request
	bidRequest, errResp := h.parser.ParseRequest(c.Request().Body, correlationID)
	if errResp != nil {
		h.logger.Warn("Bid request parsing failed",
			zap.String("correlation_id", correlationID),
			zap.String("error", errResp.Error),
		)
		h.metrics.ErrorsTotal.WithLabelValues("validation").Inc()
		return c.JSON(http.StatusBadRequest, errResp)
	}

	// Validate bid request
	validationErr := h.validator.ValidateRequest(bidRequest, correlationID)
	if validationErr != nil {
		return c.JSON(http.StatusBadRequest, validationErr)
	}

	// Log successful validation
	h.logger.Info("Bid request validated successfully",
		zap.String("correlation_id", correlationID),
		zap.String("request_id", bidRequest.ID),
		zap.Int("impressions", len(bidRequest.Imp)),
	)

	// For User Story 1, we just return a simple response acknowledging validation
	// Actual bid generation will be implemented in User Story 2 (Phase 4)
	response := models.BidResponse{
		ID:      bidRequest.ID,
		SeatBid: []models.SeatBid{}, // Empty for now - no bids generated yet
		Cur:     "USD",
	}

	return c.JSON(http.StatusOK, response)
}
