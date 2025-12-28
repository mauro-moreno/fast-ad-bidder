package models

// Campaign represents an advertising campaign with targeting and bidding rules
// This model extends the store.Campaign with additional runtime fields
type Campaign struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Status      string                 `json:"status"` // active, paused, archived
	DailyBudget float64                `json:"daily_budget"`
	SpentToday  float64                `json:"spent_today"`
	Targeting   map[string]interface{} `json:"targeting"`
	BidStrategy string                 `json:"bid_strategy"` // fixed_cpm, dynamic
	CreativeIDs []string               `json:"creative_ids"`
	BidFloorCPM float64                `json:"bid_floor_cpm"`
	MaxBidCPM   float64                `json:"max_bid_cpm"`
}

// IsActive returns true if the campaign can participate in bidding
func (c *Campaign) IsActive() bool {
	return c.Status == "active"
}

// HasBudget returns true if the campaign has remaining daily budget
func (c *Campaign) HasBudget() bool {
	return c.SpentToday < c.DailyBudget
}

// RemainingBudget returns the amount of budget left for today
func (c *Campaign) RemainingBudget() float64 {
	return c.DailyBudget - c.SpentToday
}

// BudgetUtilization returns the percentage of daily budget spent (0.0 to 1.0)
func (c *Campaign) BudgetUtilization() float64 {
	if c.DailyBudget == 0 {
		return 1.0
	}
	return c.SpentToday / c.DailyBudget
}
