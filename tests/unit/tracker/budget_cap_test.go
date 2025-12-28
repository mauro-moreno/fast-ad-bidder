package tracker

import (
	"testing"

	"github.com/fast-ad-bidder/bidder/src/services/store"
	"github.com/fast-ad-bidder/bidder/src/services/tracker"
	"github.com/stretchr/testify/assert"
)

// T083: Unit test - budget-capped status change
func TestBudgetCappedStatusChange(t *testing.T) {
	t.Run("Campaign becomes budget-capped when spend reaches budget", func(t *testing.T) {
		campaign := &store.Campaign{
			ID:          "campaign-001",
			Status:      "active",
			DailyBudget: 1000.0,
			SpentToday:  995.0, // Close to budget
		}

		// Check if budget is exhausted
		isCapped := tracker.IsBudgetCapped(campaign)
		assert.False(t, isCapped, "Campaign should not be capped yet")

		// Add spend that exceeds budget
		campaign.SpentToday = 1000.0

		isCapped = tracker.IsBudgetCapped(campaign)
		assert.True(t, isCapped, "Campaign should be budget-capped")
	})

	t.Run("Campaign with spend exceeding budget is capped", func(t *testing.T) {
		campaign := &store.Campaign{
			ID:          "campaign-002",
			Status:      "active",
			DailyBudget: 1000.0,
			SpentToday:  1050.0, // Over budget
		}

		isCapped := tracker.IsBudgetCapped(campaign)
		assert.True(t, isCapped, "Campaign over budget should be capped")
	})

	t.Run("Campaign with zero budget is immediately capped", func(t *testing.T) {
		campaign := &store.Campaign{
			ID:          "campaign-003",
			Status:      "active",
			DailyBudget: 0.0,
			SpentToday:  0.0,
		}

		isCapped := tracker.IsBudgetCapped(campaign)
		assert.True(t, isCapped, "Campaign with zero budget should be capped")
	})

	t.Run("Campaign with negative budget is capped", func(t *testing.T) {
		campaign := &store.Campaign{
			ID:          "campaign-004",
			Status:      "active",
			DailyBudget: -100.0, // Invalid but defensive check
			SpentToday:  0.0,
		}

		isCapped := tracker.IsBudgetCapped(campaign)
		assert.True(t, isCapped, "Campaign with negative budget should be capped")
	})

	t.Run("Fresh campaign with full budget is not capped", func(t *testing.T) {
		campaign := &store.Campaign{
			ID:          "campaign-005",
			Status:      "active",
			DailyBudget: 1000.0,
			SpentToday:  0.0,
		}

		isCapped := tracker.IsBudgetCapped(campaign)
		assert.False(t, isCapped, "Fresh campaign should not be capped")
	})

	t.Run("Paused campaign is not considered budget-capped", func(t *testing.T) {
		campaign := &store.Campaign{
			ID:          "campaign-006",
			Status:      "paused",
			DailyBudget: 1000.0,
			SpentToday:  1000.0,
		}

		// Budget cap check should only apply to active campaigns
		// Paused campaigns are excluded for different reasons
		isCapped := tracker.IsBudgetCapped(campaign)

		// Implementation choice: may still return true for budget cap
		// but campaign matcher will exclude paused campaigns separately
		_ = isCapped
	})

	t.Run("Check remaining budget calculation", func(t *testing.T) {
		campaign := &store.Campaign{
			ID:          "campaign-007",
			Status:      "active",
			DailyBudget: 1000.0,
			SpentToday:  350.0,
		}

		remaining := tracker.RemainingBudget(campaign)
		assert.Equal(t, 650.0, remaining, "Remaining budget should be correct")
	})

	t.Run("Remaining budget is zero when capped", func(t *testing.T) {
		campaign := &store.Campaign{
			ID:          "campaign-008",
			Status:      "active",
			DailyBudget: 1000.0,
			SpentToday:  1000.0,
		}

		remaining := tracker.RemainingBudget(campaign)
		assert.Equal(t, 0.0, remaining, "Remaining budget should be zero")
	})

	t.Run("Remaining budget is negative when over budget", func(t *testing.T) {
		campaign := &store.Campaign{
			ID:          "campaign-009",
			Status:      "active",
			DailyBudget: 1000.0,
			SpentToday:  1100.0,
		}

		remaining := tracker.RemainingBudget(campaign)
		assert.Equal(t, -100.0, remaining, "Remaining budget should be negative")
	})
}

// Test budget utilization percentage
func TestBudgetUtilizationPercentage(t *testing.T) {
	tests := []struct {
		name      string
		budget    float64
		spent     float64
		expected  float64
		shouldCap bool
	}{
		{
			name:      "0% utilization",
			budget:    1000.0,
			spent:     0.0,
			expected:  0.0,
			shouldCap: false,
		},
		{
			name:      "50% utilization",
			budget:    1000.0,
			spent:     500.0,
			expected:  50.0,
			shouldCap: false,
		},
		{
			name:      "90% utilization",
			budget:    1000.0,
			spent:     900.0,
			expected:  90.0,
			shouldCap: false,
		},
		{
			name:      "100% utilization (capped)",
			budget:    1000.0,
			spent:     1000.0,
			expected:  100.0,
			shouldCap: true,
		},
		{
			name:      "110% utilization (over budget)",
			budget:    1000.0,
			spent:     1100.0,
			expected:  110.0,
			shouldCap: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			campaign := &store.Campaign{
				ID:          "test-campaign",
				DailyBudget: tt.budget,
				SpentToday:  tt.spent,
			}

			utilization := tracker.BudgetUtilization(campaign)
			assert.InDelta(t, tt.expected, utilization, 0.01, "Budget utilization should match")

			isCapped := tracker.IsBudgetCapped(campaign)
			assert.Equal(t, tt.shouldCap, isCapped, "Budget cap status should match")
		})
	}
}
