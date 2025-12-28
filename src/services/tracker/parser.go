package tracker

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/fast-ad-bidder/bidder/src/models"
	"go.uber.org/zap"
)

// Parser handles parsing of win notification query parameters
// T091: Implement win notification parser
type Parser struct {
	logger *zap.Logger
}

// NewParser creates a new win notification parser
func NewParser(logger *zap.Logger) *Parser {
	return &Parser{
		logger: logger,
	}
}

// ParseWinNotification parses win notification from query parameters
// Expected parameters: bid, price, currency (optional)
func (p *Parser) ParseWinNotification(queryParams url.Values, correlationID string) (*models.WinNotification, *models.ErrorResponse) {
	// Extract bid ID
	bidID := queryParams.Get("bid")
	if bidID == "" {
		p.logger.Warn("Win notification missing bid ID",
			zap.String("correlation_id", correlationID),
		)
		return nil, models.NewErrorResponse("Missing bid parameter", models.ErrorCodeInvalidWinNotification).
			WithCorrelationID(correlationID)
	}

	// Extract price
	priceStr := queryParams.Get("price")
	if priceStr == "" {
		p.logger.Warn("Win notification missing price",
			zap.String("correlation_id", correlationID),
			zap.String("bid_id", bidID),
		)
		return nil, models.NewErrorResponse("Missing price parameter", models.ErrorCodeInvalidWinNotification).
			WithCorrelationID(correlationID)
	}

	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		p.logger.Warn("Win notification invalid price format",
			zap.String("correlation_id", correlationID),
			zap.String("bid_id", bidID),
			zap.String("price", priceStr),
			zap.Error(err),
		)
		return nil, models.NewErrorResponse(
			fmt.Sprintf("Invalid price format: %s", priceStr),
			models.ErrorCodeInvalidWinNotification,
		).WithCorrelationID(correlationID)
	}

	if price <= 0 {
		p.logger.Warn("Win notification invalid price value",
			zap.String("correlation_id", correlationID),
			zap.String("bid_id", bidID),
			zap.Float64("price", price),
		)
		return nil, models.NewErrorResponse(
			"Price must be positive",
			models.ErrorCodeInvalidWinNotification,
		).WithCorrelationID(correlationID)
	}

	// Extract currency (optional, defaults to USD)
	currency := queryParams.Get("currency")
	if currency == "" {
		currency = "USD"
	}

	// Create win notification
	winNotif := &models.WinNotification{
		BidID:     bidID,
		Price:     price,
		Currency:  currency,
		Timestamp: time.Now(),
	}

	p.logger.Debug("Win notification parsed successfully",
		zap.String("correlation_id", correlationID),
		zap.String("bid_id", bidID),
		zap.Float64("price", price),
		zap.String("currency", currency),
	)

	return winNotif, nil
}
