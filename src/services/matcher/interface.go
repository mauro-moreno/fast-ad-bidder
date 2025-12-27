package matcher

import (
	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/store"
)

// Matcher defines the interface for campaign matching
// Full implementation will be provided in User Story 2 (Phase 4)
type Matcher interface {
	// MatchCampaigns finds campaigns that match the bid request
	MatchCampaigns(req *models.BidRequest) ([]*store.Campaign, error)
}
