package models

import (
	"time"
)

// BidMetrics represents aggregated metrics for a campaign
// T089: Define BidMetrics model
type BidMetrics struct {
	CampaignID      string    `json:"campaign_id"`
	TotalBids       int       `json:"total_bids"`
	TotalWins       int       `json:"total_wins"`
	WinRate         float64   `json:"win_rate"`
	AvgCPM          float64   `json:"avg_cpm"`
	TotalSpend      float64   `json:"total_spend"`
	ImpressionCount int       `json:"impression_count"`
	Period          string    `json:"period"` // e.g., "hourly", "daily"
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
}

// BidEvent represents a single bid event for metrics tracking
type BidEvent struct {
	CampaignID string    `json:"campaign_id"`
	BidID      string    `json:"bid_id"`
	Price      float64   `json:"price"`
	Timestamp  time.Time `json:"timestamp"`
}

// WinEvent represents a single win event for metrics tracking
type WinEvent struct {
	CampaignID string    `json:"campaign_id"`
	BidID      string    `json:"bid_id"`
	WinPrice   float64   `json:"win_price"`
	Timestamp  time.Time `json:"timestamp"`
}

// CalculateWinRate computes the win rate percentage
func (m *BidMetrics) CalculateWinRate() {
	if m.TotalBids > 0 {
		m.WinRate = float64(m.TotalWins) / float64(m.TotalBids)
	} else {
		m.WinRate = 0.0
	}
}

// CalculateAvgCPM computes the average CPM from total spend and wins
func (m *BidMetrics) CalculateAvgCPM() {
	if m.TotalWins > 0 {
		m.AvgCPM = m.TotalSpend / float64(m.TotalWins)
	} else {
		m.AvgCPM = 0.0
	}
}
