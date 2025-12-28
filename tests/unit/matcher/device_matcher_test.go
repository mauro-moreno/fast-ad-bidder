package matcher

import (
	"testing"

	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/matcher"
	"github.com/fast-ad-bidder/bidder/src/services/store"
	"github.com/stretchr/testify/assert"
)

// T055: Unit test - campaign device type matcher
func TestDeviceTypeMatcher(t *testing.T) {
	deviceMatcher := matcher.NewDeviceMatcher()

	tests := []struct {
		name        string
		campaign    *store.Campaign
		device      *models.Device
		shouldMatch bool
	}{
		{
			name: "Match - Device type in list",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"device_types": []interface{}{float64(1), float64(2)}, // Mobile and Desktop
				},
			},
			device: &models.Device{
				DeviceType: 2, // Desktop
			},
			shouldMatch: true,
		},
		{
			name: "No Match - Device type not in list",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"device_types": []interface{}{float64(1)}, // Mobile only
				},
			},
			device: &models.Device{
				DeviceType: 2, // Desktop
			},
			shouldMatch: false,
		},
		{
			name: "Match - No device type targeting (matches all)",
			campaign: &store.Campaign{
				ID:        "test-campaign",
				Targeting: map[string]interface{}{},
			},
			device: &models.Device{
				DeviceType: 3,
			},
			shouldMatch: true,
		},
		{
			name: "Match - Tablet device type",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"device_types": []interface{}{float64(4), float64(5)}, // Tablet and Phone
				},
			},
			device: &models.Device{
				DeviceType: 5, // Phone
			},
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := deviceMatcher.Matches(tt.campaign, tt.device)
			assert.Equal(t, tt.shouldMatch, matches, "Device type match result mismatch")
		})
	}
}
