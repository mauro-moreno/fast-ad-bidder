package models

import (
	"fmt"
	"time"
)

// WinNotification represents a win notification from the ad exchange
// T088: Define WinNotification model
type WinNotification struct {
	BidID      string    `json:"bid_id"`
	Price      float64   `json:"price"`
	Currency   string    `json:"currency"`
	Timestamp  time.Time `json:"timestamp"`
	ExchangeID string    `json:"exchange_id,omitempty"`
}

// Validate checks if the win notification has all required fields
func (w *WinNotification) Validate() error {
	if w.BidID == "" {
		return fmt.Errorf("missing bid ID")
	}

	if w.Price <= 0 {
		return fmt.Errorf("invalid price: %f", w.Price)
	}

	if w.Currency == "" {
		w.Currency = "USD" // Default to USD if not specified
	}

	return nil
}

// Add error code for win notifications
const (
	ErrorCodeInvalidWinNotification = "INVALID_WIN_NOTIFICATION"
	ErrorCodeBidNotFound            = "BID_NOT_FOUND"
)
