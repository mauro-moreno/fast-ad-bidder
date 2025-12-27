package models

import (
	"github.com/prebid/openrtb/v19/openrtb2"
)

// BidRequest wraps the OpenRTB 2.5 BidRequest
// Using Prebid's official OpenRTB library for compliance
type BidRequest = openrtb2.BidRequest

// Impression wraps the OpenRTB 2.5 Impression object
type Impression = openrtb2.Imp

// Site wraps the OpenRTB 2.5 Site object
type Site = openrtb2.Site

// App wraps the OpenRTB 2.5 App object
type App = openrtb2.App

// Device wraps the OpenRTB 2.5 Device object
type Device = openrtb2.Device

// User wraps the OpenRTB 2.5 User object
type User = openrtb2.User

// Banner wraps the OpenRTB 2.5 Banner object
type Banner = openrtb2.Banner

// Geo wraps the OpenRTB 2.5 Geo object
type Geo = openrtb2.Geo
