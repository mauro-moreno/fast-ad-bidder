package matcher

import (
	"testing"

	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/matcher"
	"github.com/fast-ad-bidder/bidder/src/services/store"
	"github.com/stretchr/testify/assert"
)

// T054: Unit test - campaign geo targeting matcher
func TestGeoTargetingMatcher(t *testing.T) {
	geoMatcher := matcher.NewGeoMatcher()

	tests := []struct {
		name        string
		campaign    *store.Campaign
		device      *models.Device
		shouldMatch bool
	}{
		{
			name: "Match - Country in whitelist",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"geo": map[string]interface{}{
						"countries": []interface{}{"USA", "CAN"},
					},
				},
			},
			device: &models.Device{
				Geo: &models.Geo{
					Country: "USA",
				},
			},
			shouldMatch: true,
		},
		{
			name: "No Match - Country not in whitelist",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"geo": map[string]interface{}{
						"countries": []interface{}{"USA", "CAN"},
					},
				},
			},
			device: &models.Device{
				Geo: &models.Geo{
					Country: "GBR",
				},
			},
			shouldMatch: false,
		},
		{
			name: "Match - Country and Region match",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"geo": map[string]interface{}{
						"countries": []interface{}{"USA"},
						"regions":   []interface{}{"CA", "NY"},
					},
				},
			},
			device: &models.Device{
				Geo: &models.Geo{
					Country: "USA",
					Region:  "CA",
				},
			},
			shouldMatch: true,
		},
		{
			name: "No Match - Country matches but region doesn't",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"geo": map[string]interface{}{
						"countries": []interface{}{"USA"},
						"regions":   []interface{}{"CA", "NY"},
					},
				},
			},
			device: &models.Device{
				Geo: &models.Geo{
					Country: "USA",
					Region:  "TX",
				},
			},
			shouldMatch: false,
		},
		{
			name: "Match - No geo targeting (matches all)",
			campaign: &store.Campaign{
				ID:        "test-campaign",
				Targeting: map[string]interface{}{},
			},
			device: &models.Device{
				Geo: &models.Geo{
					Country: "ANY",
				},
			},
			shouldMatch: true,
		},
		{
			name: "No Match - Missing device geo",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"geo": map[string]interface{}{
						"countries": []interface{}{"USA"},
					},
				},
			},
			device: &models.Device{
				// No Geo field
			},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := geoMatcher.Matches(tt.campaign, tt.device)
			assert.Equal(t, tt.shouldMatch, matches, "Geo match result mismatch")
		})
	}
}
