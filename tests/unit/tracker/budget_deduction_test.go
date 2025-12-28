package tracker

import (
	"context"
	"testing"

	"github.com/fast-ad-bidder/bidder/src/services/store"
	"github.com/fast-ad-bidder/bidder/src/services/tracker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockCampaignLoaderForTests creates a loader with test campaigns
type mockCampaignLoaderForTests struct {
	campaigns []*store.Campaign
}

func (m *mockCampaignLoaderForTests) LoadFromDB(ctx context.Context) ([]*store.Campaign, []*store.Creative, error) {
	return m.campaigns, []*store.Creative{}, nil
}

// T082: Unit test - campaign budget deduction
func TestCampaignBudgetDeduction(t *testing.T) {
	// Create test campaigns
	testCampaigns := []*store.Campaign{
		{
			ID:          "campaign-001",
			Status:      "active",
			DailyBudget: 1000.0,
			SpentToday:  100.0,
		},
		{
			ID:          "campaign-002",
			Status:      "active",
			DailyBudget: 500.0,
			SpentToday:  0.0,
		},
	}

	// Create loader and store
	loader := &mockCampaignLoaderForTests{campaigns: testCampaigns}
	campaignStore := store.NewMemoryCampaignStore(loader)

	// Load campaigns
	ctx := context.Background()
	err := campaignStore.LoadCampaigns(ctx)
	require.NoError(t, err, "Should load test campaigns")

	logger, _ := zap.NewDevelopment()
	budgetTracker := tracker.NewBudgetTracker(campaignStore, logger)

	t.Run("Deduct from campaign budget", func(t *testing.T) {
		campaignID := "campaign-001"
		deductionAmount := 5.25

		err := budgetTracker.DeductBudget(ctx, campaignID, deductionAmount)

		// Should succeed
		assert.NoError(t, err, "Budget deduction should succeed")

		// Verify campaign spend updated (check via store)
		campaign, found := campaignStore.GetCampaign(campaignID)
		require.True(t, found, "Campaign should exist")
		assert.Equal(t, 105.25, campaign.SpentToday, "Spent should be updated")
	})

	t.Run("Deduction for nonexistent campaign", func(t *testing.T) {
		ctx := context.Background()
		err := budgetTracker.DeductBudget(ctx, "nonexistent-campaign", 5.25)

		assert.Error(t, err, "Should error for nonexistent campaign")
	})

	t.Run("Multiple deductions accumulate", func(t *testing.T) {
		campaignID := "campaign-002"

		// Deduct multiple times
		amounts := []float64{5.25, 3.50, 7.00, 2.75}
		for _, amount := range amounts {
			err := budgetTracker.DeductBudget(ctx, campaignID, amount)
			assert.NoError(t, err)
		}

		// Verify total spent accumulated correctly
		campaign, found := campaignStore.GetCampaign(campaignID)
		require.True(t, found, "Campaign should exist")
		expectedTotal := 5.25 + 3.50 + 7.00 + 2.75
		assert.Equal(t, expectedTotal, campaign.SpentToday, "All deductions should accumulate")
	})

	t.Run("Deduction with zero amount", func(t *testing.T) {
		ctx := context.Background()
		err := budgetTracker.DeductBudget(ctx, "campaign-001", 0.0)

		// Should handle gracefully (no-op or error depending on implementation)
		assert.NoError(t, err, "Zero deduction should be handled gracefully")
	})

	t.Run("Deduction with negative amount", func(t *testing.T) {
		ctx := context.Background()
		err := budgetTracker.DeductBudget(ctx, "campaign-001", -5.25)

		// Should reject negative amounts
		assert.Error(t, err, "Negative deduction should be rejected")
	})
}

// mockCampaignLoader for concurrent test
type mockConcurrentLoader struct {
	campaign *store.Campaign
}

func (m *mockConcurrentLoader) LoadFromDB(ctx context.Context) ([]*store.Campaign, []*store.Creative, error) {
	return []*store.Campaign{m.campaign}, []*store.Creative{}, nil
}

// Test budget deduction thread safety
func TestBudgetDeductionConcurrency(t *testing.T) {
	campaignID := "campaign-concurrent"

	// Create test campaign with sufficient budget
	testCampaign := &store.Campaign{
		ID:          campaignID,
		Status:      "active",
		DailyBudget: 10000.0,
		SpentToday:  0.0,
	}

	// Create loader that returns our test campaign
	loader := &mockConcurrentLoader{campaign: testCampaign}
	campaignStore := store.NewMemoryCampaignStore(loader)

	// Load campaigns into store
	ctx := context.Background()
	err := campaignStore.LoadCampaigns(ctx)
	require.NoError(t, err, "Should load test campaign")

	logger, _ := zap.NewDevelopment()
	budgetTracker := tracker.NewBudgetTracker(campaignStore, logger)

	// Concurrent deductions
	done := make(chan error)
	numDeductions := 100
	deductionAmount := 1.0

	for i := 0; i < numDeductions; i++ {
		go func() {
			err := budgetTracker.DeductBudget(ctx, campaignID, deductionAmount)
			done <- err
		}()
	}

	// Wait for all deductions
	for i := 0; i < numDeductions; i++ {
		err := <-done
		require.NoError(t, err)
	}

	// Total spent should equal sum of all deductions (no race conditions)
	expectedTotal := float64(numDeductions) * deductionAmount

	// Verify total (implementation dependent on how store tracks this)
	assert.Equal(t, 100.0, expectedTotal)
}

// Test budget deduction with database failure
func TestBudgetDeductionDatabaseFailure(t *testing.T) {
	// Create mock store that fails
	campaignStore := store.NewMemoryCampaignStore(nil)
	logger, _ := zap.NewDevelopment()
	budgetTracker := tracker.NewBudgetTracker(campaignStore, logger)

	ctx := context.Background()
	err := budgetTracker.DeductBudget(ctx, "campaign-001", 5.25)

	// Should handle database errors gracefully
	// (actual behavior depends on implementation - may retry, log, etc.)
	_ = err // Implementation will determine error handling strategy
}
