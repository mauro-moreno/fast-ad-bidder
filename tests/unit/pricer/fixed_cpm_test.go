package pricer

import (
	"testing"

	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/pricer"
	"github.com/fast-ad-bidder/bidder/src/services/store"
	"github.com/stretchr/testify/assert"
)

// T058: Unit test - fixed CPM bid price calculation
func TestFixedCPMBidPriceCalculation(t *testing.T) {
	cpmPricer := pricer.NewFixedCPMPricer()

	tests := []struct {
		name          string
		campaign      *store.Campaign
		impression    *models.Impression
		expectedPrice float64
	}{
		{
			name: "Standard CPM - Above bid floor",
			campaign: &store.Campaign{
				ID:          "campaign-001",
				BidFloorCPM: 0.50,
				MaxBidCPM:   5.00,
				BidStrategy: "fixed_cpm",
			},
			impression: &models.Impression{
				ID:       "imp-1",
				BidFloor: 0.25,
			},
			expectedPrice: 5.00, // Should use max_bid_cpm
		},
		{
			name: "CPM at impression bid floor",
			campaign: &store.Campaign{
				ID:          "campaign-002",
				BidFloorCPM: 1.00,
				MaxBidCPM:   3.00,
				BidStrategy: "fixed_cpm",
			},
			impression: &models.Impression{
				ID:       "imp-2",
				BidFloor: 2.50,
			},
			expectedPrice: 3.00, // max_bid_cpm, but should be >= impression bidfloor
		},
		{
			name: "High value impression",
			campaign: &store.Campaign{
				ID:          "campaign-003",
				BidFloorCPM: 2.00,
				MaxBidCPM:   10.00,
				BidStrategy: "fixed_cpm",
			},
			impression: &models.Impression{
				ID:       "imp-3",
				BidFloor: 5.00,
			},
			expectedPrice: 10.00, // Should use max_bid_cpm
		},
		{
			name: "Low CPM campaign",
			campaign: &store.Campaign{
				ID:          "campaign-004",
				BidFloorCPM: 0.10,
				MaxBidCPM:   0.75,
				BidStrategy: "fixed_cpm",
			},
			impression: &models.Impression{
				ID:       "imp-4",
				BidFloor: 0.05,
			},
			expectedPrice: 0.75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price := cpmPricer.CalculatePrice(tt.campaign, tt.impression)
			assert.Equal(t, tt.expectedPrice, price, "Bid price calculation mismatch")

			// Verify price meets minimum requirements
			assert.GreaterOrEqual(t, price, tt.impression.BidFloor, "Price should meet impression bid floor")
			assert.GreaterOrEqual(t, price, tt.campaign.BidFloorCPM, "Price should meet campaign bid floor")
			assert.LessOrEqual(t, price, tt.campaign.MaxBidCPM, "Price should not exceed campaign max bid")
		})
	}
}
