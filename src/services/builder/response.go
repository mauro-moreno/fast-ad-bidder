package builder

import (
	"fmt"

	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/store"
	"github.com/google/uuid"
	"github.com/prebid/openrtb/v19/openrtb3"
)

// ResponseBuilder builds OpenRTB bid responses
type ResponseBuilder struct {
	winNotificationURL string
}

// NewResponseBuilder creates a new response builder
func NewResponseBuilder(winNotificationURL string) *ResponseBuilder {
	return &ResponseBuilder{
		winNotificationURL: winNotificationURL,
	}
}

// BuildBidResponse creates an OpenRTB bid response with a winning bid
func (b *ResponseBuilder) BuildBidResponse(
	requestID string,
	impression *models.Impression,
	campaign *store.Campaign,
	creative *store.Creative,
	bidPrice float64,
) *models.BidResponse {
	bidID := uuid.New().String()

	// Build the bid object
	bid := models.Bid{
		ID:    bidID,
		ImpID: impression.ID,
		Price: bidPrice,
		AdID:  creative.ID,
		CrID:  creative.ID,
		W:     int64(creative.Width),
		H:     int64(creative.Height),
		AdM:   creative.Markup,
		NURL:  b.buildWinNotificationURL(bidID, bidPrice),
	}

	// Optional: Add advertiser domain if available
	if campaign.Targeting != nil {
		bid.ADomain = []string{} // TODO: Extract from campaign if available
	}

	// Build the seat bid
	seatBid := models.SeatBid{
		Bid: []models.Bid{bid},
	}

	// Build the bid response
	response := &models.BidResponse{
		ID:      requestID,
		SeatBid: []models.SeatBid{seatBid},
		Cur:     "USD",
	}

	return response
}

// BuildMultiBidResponse creates an OpenRTB bid response with multiple bids
func (b *ResponseBuilder) BuildMultiBidResponse(
	requestID string,
	bids []BidInfo,
) *models.BidResponse {
	if len(bids) == 0 {
		return b.BuildNoBidResponse(requestID, models.NoBidReasonNoMatchingCampaigns)
	}

	var bidObjects []models.Bid
	for _, bidInfo := range bids {
		bidID := uuid.New().String()

		bid := models.Bid{
			ID:    bidID,
			ImpID: bidInfo.Impression.ID,
			Price: bidInfo.Price,
			AdID:  bidInfo.Creative.ID,
			CrID:  bidInfo.Creative.ID,
			W:     int64(bidInfo.Creative.Width),
			H:     int64(bidInfo.Creative.Height),
			AdM:   bidInfo.Creative.Markup,
			NURL:  b.buildWinNotificationURL(bidID, bidInfo.Price),
		}

		bidObjects = append(bidObjects, bid)
	}

	seatBid := models.SeatBid{
		Bid: bidObjects,
	}

	response := &models.BidResponse{
		ID:      requestID,
		SeatBid: []models.SeatBid{seatBid},
		Cur:     "USD",
	}

	return response
}

// BuildNoBidResponse creates an OpenRTB bid response with no bids
func (b *ResponseBuilder) BuildNoBidResponse(requestID string, reason models.NoBidReason) *models.BidResponse {
	nbrCode := openrtb3.NoBidReason(reason)
	response := &models.BidResponse{
		ID:      requestID,
		SeatBid: []models.SeatBid{}, // Empty array for no bids
		NBR:     &nbrCode,
		Cur:     "USD",
	}

	return response
}

// buildWinNotificationURL creates the win notification URL with macros
func (b *ResponseBuilder) buildWinNotificationURL(bidID string, price float64) string {
	if b.winNotificationURL == "" {
		return ""
	}

	// Build URL with macros for price substitution
	// ${AUCTION_PRICE} will be replaced by the exchange with the actual winning price
	return fmt.Sprintf("%s?bid=%s&price=${AUCTION_PRICE}", b.winNotificationURL, bidID)
}

// BidInfo contains all information needed to build a bid
type BidInfo struct {
	Impression *models.Impression
	Campaign   *store.Campaign
	Creative   *store.Creative
	Price      float64
}
