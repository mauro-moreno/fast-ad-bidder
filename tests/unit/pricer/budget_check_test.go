package pricer

import (
	"testing"

	"github.com/fast-ad-bidder/bidder/src/services/pricer"
	"github.com/fast-ad-bidder/bidder/src/services/store"
	"github.com/stretchr/testify/assert"
)

// T059: Unit test - budget constraint validation
func TestBudgetConstraintValidation(t *testing.T) {
	budgetChecker := pricer.NewBudgetChecker()

	tests := []struct {
		name        string
		campaign    *store.Campaign
		bidPrice    float64
		shouldAllow bool
		description string
	}{
		{
			name: "Allow - Budget available",
			campaign: &store.Campaign{
				ID:          "campaign-001",
				DailyBudget: 1000.00,
				SpentToday:  500.00,
			},
			bidPrice:    5.00,
			shouldAllow: true,
			description: "Campaign has sufficient budget remaining",
		},
		{
			name: "Deny - Budget exhausted",
			campaign: &store.Campaign{
				ID:          "campaign-002",
				DailyBudget: 100.00,
				SpentToday:  100.00,
			},
			bidPrice:    5.00,
			shouldAllow: false,
			description: "Campaign has no budget remaining",
		},
		{
			name: "Deny - Budget would be exceeded",
			campaign: &store.Campaign{
				ID:          "campaign-003",
				DailyBudget: 100.00,
				SpentToday:  99.50,
			},
			bidPrice:    1.00,
			shouldAllow: false,
			description: "Bid would exceed remaining budget",
		},
		{
			name: "Allow - Exact budget remaining",
			campaign: &store.Campaign{
				ID:          "campaign-004",
				DailyBudget: 100.00,
				SpentToday:  95.00,
			},
			bidPrice:    5.00,
			shouldAllow: true,
			description: "Bid exactly matches remaining budget",
		},
		{
			name: "Allow - Fresh campaign",
			campaign: &store.Campaign{
				ID:          "campaign-005",
				DailyBudget: 1000.00,
				SpentToday:  0.00,
			},
			bidPrice:    10.00,
			shouldAllow: true,
			description: "New campaign with full budget",
		},
		{
			name: "Allow - Small remaining budget",
			campaign: &store.Campaign{
				ID:          "campaign-006",
				DailyBudget: 100.00,
				SpentToday:  99.00,
			},
			bidPrice:    0.50,
			shouldAllow: true,
			description: "Small bid within remaining budget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := budgetChecker.CheckBudget(tt.campaign, tt.bidPrice)
			assert.Equal(t, tt.shouldAllow, allowed, tt.description)

			// Additional validations
			if allowed {
				remainingBudget := tt.campaign.DailyBudget - tt.campaign.SpentToday
				assert.LessOrEqual(t, tt.bidPrice, remainingBudget, "Bid should not exceed remaining budget")
			}
		})
	}
}

// Test budget percentage thresholds
func TestBudgetPercentageThresholds(t *testing.T) {
	budgetChecker := pricer.NewBudgetChecker()

	campaign := &store.Campaign{
		ID:          "campaign-pacing",
		DailyBudget: 1000.00,
		SpentToday:  0.00,
	}

	// Test at various spend levels
	spendLevels := []struct {
		percentSpent float64
		bidPrice     float64
		shouldAllow  bool
	}{
		{0.00, 10.00, true},  // 0% spent
		{0.50, 10.00, true},  // 50% spent
		{0.90, 10.00, true},  // 90% spent
		{0.99, 10.00, true},  // 99% spent
		{1.00, 10.00, false}, // 100% spent (exhausted)
		{1.00, 0.01, false},  // 100% spent (no budget for even tiny bid)
	}

	for _, level := range spendLevels {
		campaign.SpentToday = campaign.DailyBudget * level.percentSpent
		allowed := budgetChecker.CheckBudget(campaign, level.bidPrice)
		assert.Equal(t, level.shouldAllow, allowed,
			"Budget check failed at %.0f%% spent with $%.2f bid",
			level.percentSpent*100, level.bidPrice)
	}
}
