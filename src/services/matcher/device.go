package matcher

import (
	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/store"
)

// DeviceMatcher matches campaigns based on device type targeting
type DeviceMatcher struct{}

// NewDeviceMatcher creates a new device type matcher
func NewDeviceMatcher() *DeviceMatcher {
	return &DeviceMatcher{}
}

// Matches returns true if the device type matches campaign targeting
func (m *DeviceMatcher) Matches(campaign *store.Campaign, device *models.Device) bool {
	// No device type targeting means match all
	deviceTypesTargeting, hasDeviceTypes := campaign.Targeting["device_types"]
	if !hasDeviceTypes {
		return true
	}

	// Device must be present
	if device == nil {
		return false
	}

	deviceTypesList, ok := deviceTypesTargeting.([]interface{})
	if !ok {
		return true // Invalid targeting format, default to match
	}

	// Check if device type is in the target list
	return matchesDeviceType(int64(device.DeviceType), deviceTypesList)
}

// matchesDeviceType checks if the device type is in the target list
func matchesDeviceType(deviceType int64, targetTypes []interface{}) bool {
	for _, targetType := range targetTypes {
		// Handle both float64 (from JSON unmarshaling) and int
		switch v := targetType.(type) {
		case float64:
			if int64(v) == deviceType {
				return true
			}
		case int:
			if int64(v) == deviceType {
				return true
			}
		case int64:
			if v == deviceType {
				return true
			}
		}
	}

	return false
}
