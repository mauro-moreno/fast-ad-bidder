package pricer

import (
	"github.com/fast-ad-bidder/bidder/src/services/store"
)

// BudgetChecker validates budget constraints for campaigns
type BudgetChecker struct{}

// NewBudgetChecker creates a new budget checker
func NewBudgetChecker() *BudgetChecker {
	return &BudgetChecker{}
}

// CheckBudget returns true if the campaign has sufficient budget for the bid
func (b *BudgetChecker) CheckBudget(campaign *store.Campaign, bidPrice float64) bool {
	// Calculate remaining budget
	remainingBudget := campaign.DailyBudget - campaign.SpentToday

	// Campaign must have enough budget for this bid
	return bidPrice <= remainingBudget
}

// HasBudgetRemaining returns true if the campaign has any budget left
func (b *BudgetChecker) HasBudgetRemaining(campaign *store.Campaign) bool {
	return campaign.SpentToday < campaign.DailyBudget
}

// RemainingBudget returns the amount of budget remaining for the campaign
func (b *BudgetChecker) RemainingBudget(campaign *store.Campaign) float64 {
	remaining := campaign.DailyBudget - campaign.SpentToday
	if remaining < 0 {
		return 0
	}
	return remaining
}

// BudgetUtilization returns the percentage of budget used (0.0 to 1.0)
func (b *BudgetChecker) BudgetUtilization(campaign *store.Campaign) float64 {
	if campaign.DailyBudget == 0 {
		return 1.0
	}
	utilization := campaign.SpentToday / campaign.DailyBudget
	if utilization > 1.0 {
		return 1.0
	}
	return utilization
}

// IsBudgetExhausted returns true if the campaign has spent its entire daily budget
func (b *BudgetChecker) IsBudgetExhausted(campaign *store.Campaign) bool {
	return campaign.SpentToday >= campaign.DailyBudget
}
