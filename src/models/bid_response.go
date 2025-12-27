package models

import (
	"github.com/prebid/openrtb/v19/openrtb2"
)

// BidResponse wraps the OpenRTB 2.5 BidResponse
type BidResponse = openrtb2.BidResponse

// SeatBid wraps the OpenRTB 2.5 SeatBid object
type SeatBid = openrtb2.SeatBid

// Bid wraps the OpenRTB 2.5 Bid object
type Bid = openrtb2.Bid

// NoBidReason represents OpenRTB 2.5 No-Bid Reason Codes
type NoBidReason int

const (
	// Standard OpenRTB No-Bid Reason Codes (Section 5.19)
	NoBidReasonUnknownError        NoBidReason = 0
	NoBidReasonTechnicalError      NoBidReason = 1
	NoBidReasonInvalidRequest      NoBidReason = 2
	NoBidReasonKnownWebSpider      NoBidReason = 3
	NoBidReasonNonHumanTraffic     NoBidReason = 4
	NoBidReasonProxyIP             NoBidReason = 5
	NoBidReasonUnsupportedDevice   NoBidReason = 6
	NoBidReasonBlockedPublisher    NoBidReason = 7
	NoBidReasonUnmatchedUser       NoBidReason = 8

	// Custom No-Bid Reason Codes (extension)
	NoBidReasonTimeout             NoBidReason = 100
	NoBidReasonNoMatchingCampaigns NoBidReason = 200
	NoBidReasonBudgetExhausted     NoBidReason = 201
)
