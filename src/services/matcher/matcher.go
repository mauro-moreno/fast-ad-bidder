package matcher

import (
	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/store"
	"github.com/fast-ad-bidder/bidder/src/services/tracker"
	"go.uber.org/zap"
)

// CampaignMatcher orchestrates all matching logic
type CampaignMatcher struct {
	geoMatcher      *GeoMatcher
	deviceMatcher   *DeviceMatcher
	domainMatcher   *DomainMatcher
	creativeMatcher *CreativeMatcher
	campaignStore   store.CampaignStore
	logger          *zap.Logger
}

// NewCampaignMatcher creates a new campaign matcher
func NewCampaignMatcher(campaignStore store.CampaignStore, logger *zap.Logger) *CampaignMatcher {
	return &CampaignMatcher{
		geoMatcher:      NewGeoMatcher(),
		deviceMatcher:   NewDeviceMatcher(),
		domainMatcher:   NewDomainMatcher(),
		creativeMatcher: NewCreativeMatcher(),
		campaignStore:   campaignStore,
		logger:          logger,
	}
}

// MatchResult represents a campaign match with its creative
type MatchResult struct {
	Campaign *store.Campaign
	Creative *store.Creative
}

// MatchCampaigns finds all campaigns that match the bid request
func (m *CampaignMatcher) MatchCampaigns(req *models.BidRequest) ([]*store.Campaign, error) {
	allCampaigns := m.campaignStore.GetAllCampaigns()
	var matches []*store.Campaign

	for _, campaign := range allCampaigns {
		if m.matchesCampaign(campaign, req) {
			matches = append(matches, campaign)
		}
	}

	m.logger.Debug("Campaign matching completed",
		zap.Int("total_campaigns", len(allCampaigns)),
		zap.Int("matched_campaigns", len(matches)),
	)

	return matches, nil
}

// MatchCampaignsWithCreatives finds campaigns and their matching creatives for each impression
func (m *CampaignMatcher) MatchCampaignsWithCreatives(req *models.BidRequest, imp *models.Impression) []MatchResult {
	allCampaigns := m.campaignStore.GetAllCampaigns()
	var results []MatchResult

	for _, campaign := range allCampaigns {
		// Check if campaign matches the request
		if !m.matchesCampaign(campaign, req) {
			continue
		}

		// Find matching creatives for this impression
		for _, creativeID := range campaign.CreativeIDs {
			creative, exists := m.campaignStore.GetCreative(creativeID)
			if !exists {
				continue
			}

			// Check if creative matches impression dimensions and is approved
			if m.creativeMatcher.Matches(creative, imp) && creative.ApprovalStatus == "approved" {
				results = append(results, MatchResult{
					Campaign: campaign,
					Creative: creative,
				})
				break // Use first matching creative for this campaign
			}
		}
	}

	m.logger.Debug("Campaign and creative matching completed",
		zap.String("impression_id", imp.ID),
		zap.Int("matches", len(results)),
	)

	return results
}

// matchesCampaign checks if a campaign matches the bid request
func (m *CampaignMatcher) matchesCampaign(campaign *store.Campaign, req *models.BidRequest) bool {
	// Campaign must be active
	if campaign.Status != "active" {
		return false
	}

	// T103: Exclude budget-capped campaigns
	if tracker.IsBudgetCapped(campaign) {
		m.logger.Debug("Campaign excluded due to budget cap",
			zap.String("campaign_id", campaign.ID),
			zap.Float64("spent", campaign.SpentToday),
			zap.Float64("budget", campaign.DailyBudget),
		)
		return false
	}

	// Check device targeting
	if req.Device == nil || !m.deviceMatcher.Matches(campaign, req.Device) {
		return false
	}

	// Check geo targeting
	if req.Device != nil && !m.geoMatcher.Matches(campaign, req.Device) {
		return false
	}

	// Check site/app domain targeting
	if !m.domainMatcher.Matches(campaign, req.Site, req.App) {
		return false
	}

	return true
}
