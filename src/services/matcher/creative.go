package matcher

import (
	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/store"
)

// CreativeMatcher matches creatives based on dimension requirements
type CreativeMatcher struct{}

// NewCreativeMatcher creates a new creative dimension matcher
func NewCreativeMatcher() *CreativeMatcher {
	return &CreativeMatcher{}
}

// Matches returns true if the creative dimensions match the impression requirements
func (m *CreativeMatcher) Matches(creative *store.Creative, impression *models.Impression) bool {
	// Impression must have banner spec
	if impression == nil || impression.Banner == nil {
		return false
	}

	banner := impression.Banner

	// Banner dimensions are optional pointers in OpenRTB
	if banner.W == nil || banner.H == nil {
		return false
	}

	// Match exact dimensions
	return creative.Width == int(*banner.W) && creative.Height == int(*banner.H)
}

// FindMatchingCreatives returns all creatives that match the impression dimensions
func (m *CreativeMatcher) FindMatchingCreatives(creatives []*store.Creative, impression *models.Impression) []*store.Creative {
	var matches []*store.Creative

	for _, creative := range creatives {
		if m.Matches(creative, impression) {
			matches = append(matches, creative)
		}
	}

	return matches
}
