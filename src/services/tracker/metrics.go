package tracker

import (
	"sync"
	"time"

	"github.com/fast-ad-bidder/bidder/src/models"
)

// MetricsAggregator aggregates bid and win metrics per campaign
// T093: Implement metrics aggregator
type MetricsAggregator struct {
	campaigns map[string]*CampaignMetrics
	mu        sync.RWMutex
}

// CampaignMetrics holds metrics for a single campaign
type CampaignMetrics struct {
	CampaignID string
	Bids       []BidRecord
	Wins       []WinRecord
}

// BidRecord represents a single bid event
type BidRecord struct {
	Price     float64
	Timestamp time.Time
}

// WinRecord represents a single win event
type WinRecord struct {
	Price     float64
	Timestamp time.Time
}

// NewMetricsAggregator creates a new metrics aggregator
func NewMetricsAggregator() *MetricsAggregator {
	return &MetricsAggregator{
		campaigns: make(map[string]*CampaignMetrics),
	}
}

// RecordBid records a bid event for a campaign
func (ma *MetricsAggregator) RecordBid(campaignID string, price float64) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	if _, exists := ma.campaigns[campaignID]; !exists {
		ma.campaigns[campaignID] = &CampaignMetrics{
			CampaignID: campaignID,
			Bids:       []BidRecord{},
			Wins:       []WinRecord{},
		}
	}

	ma.campaigns[campaignID].Bids = append(ma.campaigns[campaignID].Bids, BidRecord{
		Price:     price,
		Timestamp: time.Now(),
	})
}

// RecordBidWithTime records a bid event with a specific timestamp
func (ma *MetricsAggregator) RecordBidWithTime(campaignID string, price float64, timestamp time.Time) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	if _, exists := ma.campaigns[campaignID]; !exists {
		ma.campaigns[campaignID] = &CampaignMetrics{
			CampaignID: campaignID,
			Bids:       []BidRecord{},
			Wins:       []WinRecord{},
		}
	}

	ma.campaigns[campaignID].Bids = append(ma.campaigns[campaignID].Bids, BidRecord{
		Price:     price,
		Timestamp: timestamp,
	})
}

// RecordWin records a win event for a campaign
func (ma *MetricsAggregator) RecordWin(campaignID string, price float64) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	if _, exists := ma.campaigns[campaignID]; !exists {
		ma.campaigns[campaignID] = &CampaignMetrics{
			CampaignID: campaignID,
			Bids:       []BidRecord{},
			Wins:       []WinRecord{},
		}
	}

	ma.campaigns[campaignID].Wins = append(ma.campaigns[campaignID].Wins, WinRecord{
		Price:     price,
		Timestamp: time.Now(),
	})
}

// RecordWinWithTime records a win event with a specific timestamp
func (ma *MetricsAggregator) RecordWinWithTime(campaignID string, price float64, timestamp time.Time) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	if _, exists := ma.campaigns[campaignID]; !exists {
		ma.campaigns[campaignID] = &CampaignMetrics{
			CampaignID: campaignID,
			Bids:       []BidRecord{},
			Wins:       []WinRecord{},
		}
	}

	ma.campaigns[campaignID].Wins = append(ma.campaigns[campaignID].Wins, WinRecord{
		Price:     price,
		Timestamp: timestamp,
	})
}

// GetWinRate calculates the win rate for a campaign
func (ma *MetricsAggregator) GetWinRate(campaignID string) float64 {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	metrics, exists := ma.campaigns[campaignID]
	if !exists || len(metrics.Bids) == 0 {
		return 0.0
	}

	return float64(len(metrics.Wins)) / float64(len(metrics.Bids))
}

// GetAverageCPM calculates the average CPM for a campaign
func (ma *MetricsAggregator) GetAverageCPM(campaignID string) float64 {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	metrics, exists := ma.campaigns[campaignID]
	if !exists || len(metrics.Wins) == 0 {
		return 0.0
	}

	totalSpend := 0.0
	for _, win := range metrics.Wins {
		totalSpend += win.Price
	}

	return totalSpend / float64(len(metrics.Wins))
}

// GetTotalSpend calculates the total spend for a campaign
func (ma *MetricsAggregator) GetTotalSpend(campaignID string) float64 {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	metrics, exists := ma.campaigns[campaignID]
	if !exists {
		return 0.0
	}

	totalSpend := 0.0
	for _, win := range metrics.Wins {
		totalSpend += win.Price
	}

	return totalSpend
}

// GetMetricsSummary returns a summary of metrics for a campaign
func (ma *MetricsAggregator) GetMetricsSummary(campaignID string) models.BidMetrics {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	metrics, exists := ma.campaigns[campaignID]
	if !exists {
		return models.BidMetrics{
			CampaignID: campaignID,
		}
	}

	totalBids := len(metrics.Bids)
	totalWins := len(metrics.Wins)

	var totalSpend float64
	for _, win := range metrics.Wins {
		totalSpend += win.Price
	}

	var winRate float64
	if totalBids > 0 {
		winRate = float64(totalWins) / float64(totalBids)
	}

	var avgCPM float64
	if totalWins > 0 {
		avgCPM = totalSpend / float64(totalWins)
	}

	return models.BidMetrics{
		CampaignID: campaignID,
		TotalBids:  totalBids,
		TotalWins:  totalWins,
		WinRate:    winRate,
		AvgCPM:     avgCPM,
		TotalSpend: totalSpend,
	}
}

// GetMetricsForPeriod returns metrics for a specific time period
func (ma *MetricsAggregator) GetMetricsForPeriod(campaignID string, duration time.Duration) models.BidMetrics {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	metrics, exists := ma.campaigns[campaignID]
	if !exists {
		return models.BidMetrics{
			CampaignID: campaignID,
		}
	}

	cutoff := time.Now().Add(-duration)

	// Count bids and wins within the period
	bidCount := 0
	for _, bid := range metrics.Bids {
		if bid.Timestamp.After(cutoff) {
			bidCount++
		}
	}

	var winCount int
	var totalSpend float64
	for _, win := range metrics.Wins {
		if win.Timestamp.After(cutoff) {
			winCount++
			totalSpend += win.Price
		}
	}

	var winRate float64
	if bidCount > 0 {
		winRate = float64(winCount) / float64(bidCount)
	}

	var avgCPM float64
	if winCount > 0 {
		avgCPM = totalSpend / float64(winCount)
	}

	return models.BidMetrics{
		CampaignID: campaignID,
		TotalBids:  bidCount,
		TotalWins:  winCount,
		WinRate:    winRate,
		AvgCPM:     avgCPM,
		TotalSpend: totalSpend,
		StartTime:  cutoff,
		EndTime:    time.Now(),
	}
}

// Reset clears all metrics for a campaign
func (ma *MetricsAggregator) Reset(campaignID string) {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	delete(ma.campaigns, campaignID)
}
