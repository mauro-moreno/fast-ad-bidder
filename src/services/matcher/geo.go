package matcher

import (
	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/store"
)

// GeoMatcher matches campaigns based on geographic targeting
type GeoMatcher struct{}

// NewGeoMatcher creates a new geo targeting matcher
func NewGeoMatcher() *GeoMatcher {
	return &GeoMatcher{}
}

// Matches returns true if the device geo matches campaign targeting
func (m *GeoMatcher) Matches(campaign *store.Campaign, device *models.Device) bool {
	// No geo targeting means match all
	geoTargeting, hasGeo := campaign.Targeting["geo"]
	if !hasGeo {
		return true
	}

	// Device must have geo information
	if device == nil || device.Geo == nil {
		return false
	}

	geoMap, ok := geoTargeting.(map[string]interface{})
	if !ok {
		return true // Invalid targeting format, default to match
	}

	// Check country targeting
	if countries, hasCountries := geoMap["countries"]; hasCountries {
		countryList, ok := countries.([]interface{})
		if ok && !matchesCountry(device.Geo.Country, countryList) {
			return false
		}
	}

	// Check region targeting (only if device has region info)
	// If campaign has region targeting but device lacks region, we allow match based on country alone
	if regions, hasRegions := geoMap["regions"]; hasRegions && device.Geo.Region != "" {
		regionList, ok := regions.([]interface{})
		if ok && !matchesRegion(device.Geo.Region, regionList) {
			return false
		}
	}

	return true
}

// matchesCountry checks if the device country is in the target list
func matchesCountry(deviceCountry string, targetCountries []interface{}) bool {
	if deviceCountry == "" {
		return false
	}

	for _, country := range targetCountries {
		if countryStr, ok := country.(string); ok {
			if countryStr == deviceCountry {
				return true
			}
		}
	}

	return false
}

// matchesRegion checks if the device region is in the target list
func matchesRegion(deviceRegion string, targetRegions []interface{}) bool {
	for _, region := range targetRegions {
		if regionStr, ok := region.(string); ok {
			if regionStr == deviceRegion {
				return true
			}
		}
	}

	return false
}
