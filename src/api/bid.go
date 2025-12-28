package api

import (
	"net/http"
	"time"

	"github.com/fast-ad-bidder/bidder/src/lib"
	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/builder"
	"github.com/fast-ad-bidder/bidder/src/services/matcher"
	"github.com/fast-ad-bidder/bidder/src/services/pricer"
	"github.com/fast-ad-bidder/bidder/src/services/tracker"
	"github.com/fast-ad-bidder/bidder/src/services/validator"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// BidHandler handles OpenRTB bid requests
type BidHandler struct {
	parser          *validator.Parser
	validator       *validator.Validator
	campaignMatcher *matcher.CampaignMatcher
	pricer          *pricer.FixedCPMPricer
	budgetChecker   *pricer.BudgetChecker
	responseBuilder *builder.ResponseBuilder
	bidCache        *tracker.BidCache
	metricsAgg      *tracker.MetricsAggregator
	influxWriter    *tracker.InfluxWriter
	logger          *zap.Logger
	metrics         *lib.Metrics
}

// NewBidHandler creates a new bid handler
func NewBidHandler(
	campaignMatcher *matcher.CampaignMatcher,
	responseBuilder *builder.ResponseBuilder,
	bidCache *tracker.BidCache,
	metricsAgg *tracker.MetricsAggregator,
	influxWriter *tracker.InfluxWriter,
	logger *zap.Logger,
	metrics *lib.Metrics,
) *BidHandler {
	return &BidHandler{
		parser:          validator.NewParser(logger),
		validator:       validator.NewValidator(logger, metrics),
		campaignMatcher: campaignMatcher,
		pricer:          pricer.NewFixedCPMPricer(),
		budgetChecker:   pricer.NewBudgetChecker(),
		responseBuilder: responseBuilder,
		bidCache:        bidCache,
		metricsAgg:      metricsAgg,
		influxWriter:    influxWriter,
		logger:          logger,
		metrics:         metrics,
	}
}

// HandleBid processes incoming OpenRTB bid requests
func (h *BidHandler) HandleBid(c echo.Context) error {
	start := time.Now()

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

	// Process each impression and generate bids
	var allBids []builder.BidInfo

	for _, imp := range bidRequest.Imp {
		// Find matching campaigns and creatives for this impression
		matches := h.campaignMatcher.MatchCampaignsWithCreatives(bidRequest, &imp)

		if len(matches) == 0 {
			h.logger.Debug("No campaign matches found",
				zap.String("correlation_id", correlationID),
				zap.String("impression_id", imp.ID),
			)
			continue
		}

		// Select best bid from matches
		var bestBid *builder.BidInfo
		var bestPrice float64

		for _, match := range matches {
			// Calculate bid price
			bidPrice := h.pricer.CalculatePrice(match.Campaign, &imp)

			// Check budget constraints
			if !h.budgetChecker.CheckBudget(match.Campaign, bidPrice) {
				h.logger.Debug("Campaign budget insufficient",
					zap.String("campaign_id", match.Campaign.ID),
					zap.Float64("bid_price", bidPrice),
					zap.Float64("remaining_budget", h.budgetChecker.RemainingBudget(match.Campaign)),
				)
				continue
			}

			// Track highest bid
			if bidPrice > bestPrice {
				bestPrice = bidPrice
				bestBid = &builder.BidInfo{
					Impression: &imp,
					Campaign:   match.Campaign,
					Creative:   match.Creative,
					Price:      bidPrice,
				}

				// Log bid selection (T077)
				h.logger.Info("Bid generated",
					zap.String("correlation_id", correlationID),
					zap.String("campaign_id", match.Campaign.ID),
					zap.String("creative_id", match.Creative.ID),
					zap.Float64("bid_price", bidPrice),
					zap.String("impression_id", imp.ID),
				)
			}
		}

		if bestBid != nil {
			allBids = append(allBids, *bestBid)
		}
	}

	// Build response
	var response *models.BidResponse
	if len(allBids) > 0 {
		response = h.responseBuilder.BuildMultiBidResponse(bidRequest.ID, allBids)
		h.metrics.BidsTotal.WithLabelValues("bid").Inc()

		// T097: Add bids to cache for win notification tracking
		for _, seatBid := range response.SeatBid {
			for _, bid := range seatBid.Bid {
				// Find the corresponding bid info to get campaign ID
				for _, bidInfo := range allBids {
					if bid.ImpID == bidInfo.Impression.ID {
						// Store bid in cache with campaign info
						cachedBid := &tracker.CachedBid{
							BidID:      bid.ID,
							CampaignID: bidInfo.Campaign.ID,
							Price:      bid.Price,
							Timestamp:  time.Now(),
						}
						h.bidCache.Store(bid.ID, cachedBid)

						// Record bid in metrics aggregator
						h.metricsAgg.RecordBid(bidInfo.Campaign.ID, bid.Price)

						// Write bid event to InfluxDB
						if h.influxWriter != nil {
							bidEvent := &models.BidEvent{
								CampaignID: bidInfo.Campaign.ID,
								BidID:      bid.ID,
								Price:      bid.Price,
								Timestamp:  time.Now(),
							}
							if err := h.influxWriter.WriteBidEvent(c.Request().Context(), bidEvent); err != nil {
								h.logger.Error("Failed to write bid event to InfluxDB",
									zap.String("correlation_id", correlationID),
									zap.String("bid_id", bid.ID),
									zap.Error(err),
								)
							}
						}

						h.logger.Debug("Bid cached for win tracking",
							zap.String("correlation_id", correlationID),
							zap.String("bid_id", bid.ID),
							zap.String("campaign_id", bidInfo.Campaign.ID),
						)
						break
					}
				}
			}
		}
	} else {
		response = h.responseBuilder.BuildNoBidResponse(bidRequest.ID, models.NoBidReasonNoMatchingCampaigns)
		h.metrics.BidsTotal.WithLabelValues("nobid").Inc()
	}

	// Record latency (T076)
	duration := time.Since(start).Seconds()
	h.metrics.LatencyHistogram.WithLabelValues("/bid").Observe(duration)

	h.logger.Info("Bid response generated",
		zap.String("correlation_id", correlationID),
		zap.String("request_id", bidRequest.ID),
		zap.Int("bids_count", len(allBids)),
		zap.Duration("latency", time.Since(start)),
	)

	return c.JSON(http.StatusOK, response)
}
