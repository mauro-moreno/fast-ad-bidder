package models

import "fmt"

// Creative represents an ad creative with display specifications
type Creative struct {
	ID                string                 `json:"id"`
	CampaignID        string                 `json:"campaign_id"`
	Width             int                    `json:"width"`
	Height            int                    `json:"height"`
	Markup            string                 `json:"markup"` // HTML ad markup
	ApprovalStatus    string                 `json:"approval_status"`
	ExchangeApprovals map[string]interface{} `json:"exchange_approvals"`
}

// IsApproved returns true if the creative is approved for serving
func (c *Creative) IsApproved() bool {
	return c.ApprovalStatus == "approved"
}

// IsApprovedForExchange returns true if the creative is approved for a specific exchange
func (c *Creative) IsApprovedForExchange(exchange string) bool {
	if !c.IsApproved() {
		return false
	}

	if c.ExchangeApprovals == nil {
		return false
	}

	approval, exists := c.ExchangeApprovals[exchange]
	if !exists {
		return false
	}

	// Check if approval status is "approved"
	if approvalStr, ok := approval.(string); ok {
		return approvalStr == "approved"
	}

	return false
}

// Dimensions returns the creative dimensions as a string (e.g., "300x250")
func (c *Creative) Dimensions() string {
	return fmt.Sprintf("%dx%d", c.Width, c.Height)
}

// MatchesDimensions returns true if the creative matches the given dimensions
func (c *Creative) MatchesDimensions(width, height int64) bool {
	return c.Width == int(width) && c.Height == int(height)
}
