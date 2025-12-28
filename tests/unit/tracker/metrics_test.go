package tracker

import (
	"testing"
	"time"

	"github.com/fast-ad-bidder/bidder/src/services/tracker"
	"github.com/stretchr/testify/assert"
)

// T084: Unit test - win metrics aggregation (win rate, avg CPM)
func TestWinMetricsAggregation(t *testing.T) {
	aggregator := tracker.NewMetricsAggregator()

	t.Run("Calculate win rate", func(t *testing.T) {
		campaignID := "campaign-001"

		// Record some bid and win events
		aggregator.RecordBid(campaignID, 5.25)
		aggregator.RecordBid(campaignID, 4.50)
		aggregator.RecordBid(campaignID, 6.00)
		aggregator.RecordBid(campaignID, 3.75)

		aggregator.RecordWin(campaignID, 5.25)
		aggregator.RecordWin(campaignID, 6.00)

		// Win rate should be 2/4 = 50%
		winRate := aggregator.GetWinRate(campaignID)
		assert.InDelta(t, 0.50, winRate, 0.01, "Win rate should be 50%")
	})

	t.Run("Calculate average CPM for wins", func(t *testing.T) {
		campaignID := "campaign-002"

		// Record wins with different CPMs
		aggregator.RecordWin(campaignID, 5.00)
		aggregator.RecordWin(campaignID, 7.00)
		aggregator.RecordWin(campaignID, 3.00)

		// Average should be (5.00 + 7.00 + 3.00) / 3 = 5.00
		avgCPM := aggregator.GetAverageCPM(campaignID)
		assert.InDelta(t, 5.00, avgCPM, 0.01, "Average CPM should be 5.00")
	})

	t.Run("Calculate total spend", func(t *testing.T) {
		campaignID := "campaign-003"

		// Record wins
		aggregator.RecordWin(campaignID, 5.25)
		aggregator.RecordWin(campaignID, 4.50)
		aggregator.RecordWin(campaignID, 6.00)

		// Total spend should be sum of all win prices
		totalSpend := aggregator.GetTotalSpend(campaignID)
		assert.InDelta(t, 15.75, totalSpend, 0.01, "Total spend should be 15.75")
	})

	t.Run("Win rate with no bids is zero", func(t *testing.T) {
		campaignID := "campaign-empty"

		winRate := aggregator.GetWinRate(campaignID)
		assert.Equal(t, 0.0, winRate, "Win rate should be zero with no bids")
	})

	t.Run("Average CPM with no wins is zero", func(t *testing.T) {
		campaignID := "campaign-no-wins"

		// Record bids but no wins
		aggregator.RecordBid(campaignID, 5.00)
		aggregator.RecordBid(campaignID, 6.00)

		avgCPM := aggregator.GetAverageCPM(campaignID)
		assert.Equal(t, 0.0, avgCPM, "Average CPM should be zero with no wins")

		winRate := aggregator.GetWinRate(campaignID)
		assert.Equal(t, 0.0, winRate, "Win rate should be zero with no wins")
	})

	t.Run("Metrics for multiple campaigns", func(t *testing.T) {
		// Campaign 1: 3 bids, 2 wins
		aggregator.RecordBid("campaign-multi-1", 5.00)
		aggregator.RecordBid("campaign-multi-1", 6.00)
		aggregator.RecordBid("campaign-multi-1", 4.00)
		aggregator.RecordWin("campaign-multi-1", 5.00)
		aggregator.RecordWin("campaign-multi-1", 6.00)

		// Campaign 2: 2 bids, 1 win
		aggregator.RecordBid("campaign-multi-2", 3.50)
		aggregator.RecordBid("campaign-multi-2", 4.50)
		aggregator.RecordWin("campaign-multi-2", 3.50)

		// Campaign 1 metrics
		winRate1 := aggregator.GetWinRate("campaign-multi-1")
		assert.InDelta(t, 0.667, winRate1, 0.01, "Campaign 1 win rate")

		avgCPM1 := aggregator.GetAverageCPM("campaign-multi-1")
		assert.InDelta(t, 5.50, avgCPM1, 0.01, "Campaign 1 avg CPM")

		// Campaign 2 metrics
		winRate2 := aggregator.GetWinRate("campaign-multi-2")
		assert.InDelta(t, 0.50, winRate2, 0.01, "Campaign 2 win rate")

		avgCPM2 := aggregator.GetAverageCPM("campaign-multi-2")
		assert.InDelta(t, 3.50, avgCPM2, 0.01, "Campaign 2 avg CPM")
	})

	t.Run("Get metrics summary", func(t *testing.T) {
		campaignID := "campaign-summary"

		aggregator.RecordBid(campaignID, 5.00)
		aggregator.RecordBid(campaignID, 6.00)
		aggregator.RecordBid(campaignID, 7.00)
		aggregator.RecordWin(campaignID, 5.00)
		aggregator.RecordWin(campaignID, 7.00)

		summary := aggregator.GetMetricsSummary(campaignID)

		assert.Equal(t, 3, summary.TotalBids, "Total bids")
		assert.Equal(t, 2, summary.TotalWins, "Total wins")
		assert.InDelta(t, 0.667, summary.WinRate, 0.01, "Win rate")
		assert.InDelta(t, 6.00, summary.AvgCPM, 0.01, "Average CPM")
		assert.InDelta(t, 12.00, summary.TotalSpend, 0.01, "Total spend")
	})
}

// Test metrics time-based aggregation
func TestMetricsTimeBasedAggregation(t *testing.T) {
	aggregator := tracker.NewMetricsAggregator()
	campaignID := "campaign-time"

	t.Run("Hourly metrics aggregation", func(t *testing.T) {
		now := time.Now()

		// Record events with timestamps
		aggregator.RecordBidWithTime(campaignID, 5.00, now.Add(-30*time.Minute))
		aggregator.RecordWinWithTime(campaignID, 5.00, now.Add(-30*time.Minute))

		aggregator.RecordBidWithTime(campaignID, 6.00, now.Add(-25*time.Minute))
		aggregator.RecordWinWithTime(campaignID, 6.00, now.Add(-25*time.Minute))

		// Get metrics for last hour
		hourlyMetrics := aggregator.GetMetricsForPeriod(campaignID, time.Hour)

		assert.Equal(t, 2, hourlyMetrics.TotalBids)
		assert.Equal(t, 2, hourlyMetrics.TotalWins)
		assert.Equal(t, 1.0, hourlyMetrics.WinRate)
	})

	t.Run("Daily metrics aggregation", func(t *testing.T) {
		dailyMetrics := aggregator.GetMetricsForPeriod(campaignID, 24*time.Hour)

		// Should include all events from the day
		assert.True(t, dailyMetrics.TotalBids >= 2)
		assert.True(t, dailyMetrics.TotalWins >= 2)
	})
}

// Test metrics reset
func TestMetricsReset(t *testing.T) {
	aggregator := tracker.NewMetricsAggregator()
	campaignID := "campaign-reset"

	// Record some events
	aggregator.RecordBid(campaignID, 5.00)
	aggregator.RecordWin(campaignID, 5.00)

	// Verify metrics exist
	summary := aggregator.GetMetricsSummary(campaignID)
	assert.Equal(t, 1, summary.TotalBids)
	assert.Equal(t, 1, summary.TotalWins)

	// Reset metrics
	aggregator.Reset(campaignID)

	// Verify metrics cleared
	summary = aggregator.GetMetricsSummary(campaignID)
	assert.Equal(t, 0, summary.TotalBids)
	assert.Equal(t, 0, summary.TotalWins)
}
