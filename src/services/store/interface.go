package store

import (
	"context"
)

// Campaign represents a loaded campaign in memory
type Campaign struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Status        string                 `json:"status"`
	DailyBudget   float64                `json:"daily_budget"`
	SpentToday    float64                `json:"spent_today"`
	Targeting     map[string]interface{} `json:"targeting"`
	BidStrategy   string                 `json:"bid_strategy"`
	CreativeIDs   []string               `json:"creative_ids"`
	BidFloorCPM   float64                `json:"bid_floor_cpm"`
	MaxBidCPM     float64                `json:"max_bid_cpm"`
}

// Creative represents a creative asset
type Creative struct {
	ID                string                 `json:"id"`
	CampaignID        string                 `json:"campaign_id"`
	Width             int                    `json:"width"`
	Height            int                    `json:"height"`
	Markup            string                 `json:"markup"`
	ApprovalStatus    string                 `json:"approval_status"`
	ExchangeApprovals map[string]interface{} `json:"exchange_approvals"`
}

// CampaignStore defines the interface for campaign storage and retrieval
type CampaignStore interface {
	// LoadCampaigns loads all active campaigns into memory
	LoadCampaigns(ctx context.Context) error

	// GetCampaign retrieves a campaign by ID from memory
	GetCampaign(campaignID string) (*Campaign, bool)

	// GetAllCampaigns returns all active campaigns from memory
	GetAllCampaigns() []*Campaign

	// GetCreative retrieves a creative by ID
	GetCreative(creativeID string) (*Creative, bool)

	// UpdateSpent updates the daily spend for a campaign
	UpdateSpent(campaignID string, amount float64) error

	// RefreshCampaigns reloads campaigns from the database
	RefreshCampaigns(ctx context.Context) error

	// Close closes any open connections
	Close() error
}
