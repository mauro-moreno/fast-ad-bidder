package pricer

import (
	"math"

	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/store"
)

// FixedCPMPricer calculates bid prices using a fixed CPM strategy
type FixedCPMPricer struct{}

// NewFixedCPMPricer creates a new fixed CPM pricer
func NewFixedCPMPricer() *FixedCPMPricer {
	return &FixedCPMPricer{}
}

// CalculatePrice calculates the bid price for a fixed CPM campaign
func (p *FixedCPMPricer) CalculatePrice(campaign *store.Campaign, impression *models.Impression) float64 {
	// Start with campaign's max bid CPM
	price := campaign.MaxBidCPM

	// Ensure price meets campaign's own bid floor
	if price < campaign.BidFloorCPM {
		price = campaign.BidFloorCPM
	}

	// Ensure price meets impression's bid floor
	if impression.BidFloor > 0 && price < impression.BidFloor {
		price = impression.BidFloor
	}

	// Cap at campaign's max bid
	if price > campaign.MaxBidCPM {
		price = campaign.MaxBidCPM
	}

	// Round to 2 decimal places (CPM precision)
	price = math.Round(price*100) / 100

	return price
}

// CalculatePriceWithBudget calculates bid price considering remaining budget
func (p *FixedCPMPricer) CalculatePriceWithBudget(campaign *store.Campaign, impression *models.Impression) float64 {
	basePrice := p.CalculatePrice(campaign, impression)

	// Check if campaign has enough budget for this bid
	remainingBudget := campaign.DailyBudget - campaign.SpentToday
	if basePrice > remainingBudget {
		// Reduce price to match remaining budget
		basePrice = remainingBudget
		basePrice = math.Round(basePrice*100) / 100
	}

	return basePrice
}
