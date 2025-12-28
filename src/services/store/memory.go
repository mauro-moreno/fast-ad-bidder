package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// T099: Prometheus gauges for active campaigns and budget tracking
	activeCampaignsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "active_campaigns_total",
		Help: "Total number of active campaigns with available budget",
	})

	totalBudgetGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "campaigns_budget_total",
		Help: "Total daily budget across all active campaigns",
	})

	remainingBudgetGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "campaigns_budget_remaining",
		Help: "Total remaining budget across all active campaigns",
	})

	spentBudgetGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "campaigns_budget_spent",
		Help: "Total spent budget across all active campaigns today",
	})

	budgetUtilizationGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "campaigns_budget_utilization",
		Help: "Average budget utilization percentage across all active campaigns",
	})
)

// MemoryCampaignStore implements CampaignStore using in-memory maps with RWMutex
type MemoryCampaignStore struct {
	campaigns map[string]*Campaign
	creatives map[string]*Creative
	mu        sync.RWMutex

	// loader is used to fetch campaigns from persistent storage
	loader CampaignLoader
}

// CampaignLoader defines the interface for loading campaigns from persistent storage
type CampaignLoader interface {
	LoadFromDB(ctx context.Context) ([]*Campaign, []*Creative, error)
}

// NewMemoryCampaignStore creates a new in-memory campaign store
func NewMemoryCampaignStore(loader CampaignLoader) *MemoryCampaignStore {
	return &MemoryCampaignStore{
		campaigns: make(map[string]*Campaign),
		creatives: make(map[string]*Creative),
		loader:    loader,
	}
}

// LoadCampaigns loads all active campaigns into memory
func (s *MemoryCampaignStore) LoadCampaigns(ctx context.Context) error {
	if s.loader == nil {
		return fmt.Errorf("no campaign loader configured")
	}

	campaigns, creatives, err := s.loader.LoadFromDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to load campaigns: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Replace existing campaigns
	s.campaigns = make(map[string]*Campaign)
	for _, campaign := range campaigns {
		s.campaigns[campaign.ID] = campaign
	}

	// Replace existing creatives
	s.creatives = make(map[string]*Creative)
	for _, creative := range creatives {
		s.creatives[creative.ID] = creative
	}

	// Update Prometheus metrics
	s.updateMetrics()

	return nil
}

// GetCampaign retrieves a campaign by ID from memory
func (s *MemoryCampaignStore) GetCampaign(campaignID string) (*Campaign, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	campaign, exists := s.campaigns[campaignID]
	return campaign, exists
}

// GetAllCampaigns returns all active campaigns from memory
func (s *MemoryCampaignStore) GetAllCampaigns() []*Campaign {
	s.mu.RLock()
	defer s.mu.RUnlock()

	campaigns := make([]*Campaign, 0, len(s.campaigns))
	for _, campaign := range s.campaigns {
		// Only return active campaigns that haven't exceeded budget
		if campaign.Status == "active" && campaign.SpentToday < campaign.DailyBudget {
			campaigns = append(campaigns, campaign)
		}
	}

	return campaigns
}

// GetCreative retrieves a creative by ID
func (s *MemoryCampaignStore) GetCreative(creativeID string) (*Creative, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	creative, exists := s.creatives[creativeID]
	return creative, exists
}

// UpdateSpent updates the daily spend for a campaign
func (s *MemoryCampaignStore) UpdateSpent(campaignID string, amount float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	campaign, exists := s.campaigns[campaignID]
	if !exists {
		return fmt.Errorf("campaign not found: %s", campaignID)
	}

	campaign.SpentToday += amount

	// Update Prometheus metrics after budget change
	s.updateMetrics()

	return nil
}

// RefreshCampaigns reloads campaigns from the database
func (s *MemoryCampaignStore) RefreshCampaigns(ctx context.Context) error {
	return s.LoadCampaigns(ctx)
}

// Close closes any open connections (no-op for memory store)
func (s *MemoryCampaignStore) Close() error {
	return nil
}

// updateMetrics updates Prometheus gauges with current campaign metrics
// T099: Update Prometheus gauges for monitoring
func (s *MemoryCampaignStore) updateMetrics() {
	var activeCount int
	var totalBudget, remainingBudget, spentBudget float64
	var utilizationSum float64

	for _, campaign := range s.campaigns {
		if campaign.Status == "active" {
			totalBudget += campaign.DailyBudget
			spentBudget += campaign.SpentToday
			remaining := campaign.DailyBudget - campaign.SpentToday
			if remaining > 0 {
				activeCount++
				remainingBudget += remaining
			}

			// Calculate utilization percentage for this campaign
			if campaign.DailyBudget > 0 {
				utilization := (campaign.SpentToday / campaign.DailyBudget) * 100.0
				utilizationSum += utilization
			}
		}
	}

	// Update gauges
	activeCampaignsGauge.Set(float64(activeCount))
	totalBudgetGauge.Set(totalBudget)
	remainingBudgetGauge.Set(remainingBudget)
	spentBudgetGauge.Set(spentBudget)

	// Calculate average utilization
	if activeCount > 0 {
		avgUtilization := utilizationSum / float64(len(s.campaigns))
		budgetUtilizationGauge.Set(avgUtilization)
	} else {
		budgetUtilizationGauge.Set(0)
	}
}
