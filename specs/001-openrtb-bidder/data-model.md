# Data Model: OpenRTB Bidder

**Feature**: OpenRTB Bidder (Google Services Integration)
**Date**: 2025-12-27
**Purpose**: Define entity schemas and relationships for bidder implementation

## Overview

This data model supports three user stories:
- **P1**: Request validation (BidRequest, BidResponse entities)
- **P2**: Bid generation (Campaign, Creative, Impression entities)
- **P3**: Metrics tracking (WinNotification, BidMetrics entities)

All entities follow OpenRTB 2.5 specification where applicable.

---

## Core Entities

### BidRequest

Incoming auction opportunity from Google Ad Exchange.

**Source**: OpenRTB 2.5 BidRequest object (via Prebid openrtb2.BidRequest)

**Key Fields**:
```go
type BidRequest struct {
    ID          string         // Unique auction ID
    Imp         []Impression   // Array of impression opportunities
    Site        *Site          // Site details (for display ads) - exclusive with App
    App         *App           // App details (for mobile apps) - exclusive with Site
    Device      *Device        // User device information
    User        *User          // User demographic data (optional)
    TMax        int            // Auction timeout in milliseconds (from Google)
    WSeat       []string       // Allowed buyer seats (whitelist)
    BSeat       []string       // Blocked buyer seats (blacklist)
    Currency    []string       // Allowed currencies (default: ["USD"])
    Ext         json.RawMessage // OpenRTB extensions
}
```

**Validation Rules** (from FR-002):
- `ID` MUST be present and non-empty
- `Imp` array MUST contain at least one impression
- Either `Site` OR `App` MUST be present (not both)
- `Device` MUST be present

**Relationships**:
- Contains 1..N `Impression` objects
- References optional `Site` or `App`
- References `Device` information

**Storage**: Not persisted (ephemeral, processed in-memory during request handling)

**OpenRTB Reference**: https://github.com/prebid/openrtb/blob/master/openrtb2/bidrequest.go

---

### Impression

Individual ad placement opportunity within a bid request.

**Source**: OpenRTB 2.5 Imp object

**Key Fields**:
```go
type Impression struct {
    ID              string     // Unique impression ID within request
    Banner          *Banner    // Banner ad specifications (for this feature scope)
    DisplayManager  string     // Rendering entity (e.g., "Google AdX")
    DisplayMgrVer   string     // Version of display manager
    Instl           int        // 1 = interstitial/full-screen, 0 = normal
    TagID           string     // Ad tag ID from publisher
    BidFloor        float64    // Minimum CPM bid price (USD)
    BidFloorCur     string     // Currency for bid floor (default: "USD")
    Secure          *int       // 1 = HTTPS required, 0 = HTTP allowed
    Ext             json.RawMessage
}

type Banner struct {
    W      *int    // Width in pixels (e.g., 300)
    H      *int    // Height in pixels (e.g., 250)
    WMax   int     // Maximum width
    HMax   int     // Maximum height
    WMin   int     // Minimum width
    HMin   int     // Minimum height
    Format []Format // Array of supported sizes
    Pos    int      // Ad position (0 = unknown, 1-7 = specific positions)
}

type Format struct {
    W int // Width in pixels
    H int // Height in pixels
}
```

**Validation Rules** (from FR-003):
- `ID` MUST be unique within request
- `Banner` MUST be present (banner-only scope for this feature)
- At least one dimension (W/H or Format array) MUST be specified
- `BidFloor` MUST be non-negative

**Targeting Extraction** (from FR-003):
- Dimensions: Extract from Banner.W/H or Banner.Format array
- Position: Extract from Banner.Pos
- Secure requirement: Extract from Secure field

**Relationships**:
- Belongs to one `BidRequest`
- Matches against `Creative` dimensions

---

### BidResponse

Outgoing response containing zero or more bids.

**Source**: OpenRTB 2.5 BidResponse object

**Key Fields**:
```go
type BidResponse struct {
    ID      string     // Must match BidRequest.ID
    SeatBid []SeatBid  // Array of seat bids (typically one for single bidder)
    BidID   string     // Bidder-generated response ID for tracking
    Cur     string     // Currency (default: "USD")
    NBR     *int       // No-bid reason code (if SeatBid is empty)
    Ext     json.RawMessage
}

type SeatBid struct {
    Bid   []Bid   // Array of bids (one per impression)
    Seat  string  // Seat ID (bidder identifier)
    Group int     // 0 = impressions can be won individually, 1 = all-or-nothing
    Ext   json.RawMessage
}

type Bid struct {
    ID       string   // Unique bid ID
    ImpID    string   // Matches Impression.ID from request
    Price    float64  // Bid price in CPM (USD)
    AdID     string   // Ad creative ID
    NURL     string   // Win notification URL (optional)
    ADomain  []string // Advertiser domains (e.g., ["example.com"])
    IURL     string   // Image URL for ad content (optional)
    CID      string   // Campaign ID
    CrID     string   // Creative ID
    W        int      // Creative width (must match impression)
    H        int      // Creative height (must match impression)
    Ext      json.RawMessage
}
```

**Generation Rules** (from FR-006):
- `ID` MUST match incoming BidRequest.ID
- `Bid.ImpID` MUST match valid impression from request
- `Bid.Price` calculated by pricing service (see Campaign entity)
- `Bid.ADomain` extracted from Creative.AdvertiserDomain
- `Bid.W` and `Bid.H` MUST match impression dimensions

**No-Bid Responses** (from FR-008):
- Empty `SeatBid` array indicates no bid
- Optional `NBR` field provides reason code (e.g., 2 = "no matching campaigns")

**Storage**: Not persisted directly (logged for debugging, bid ID tracked in WinNotification)

---

### Campaign

Advertiser campaign configuration for targeting and bidding.

**Purpose**: Defines targeting criteria, budget constraints, and bid strategy

**Schema**:
```go
type Campaign struct {
    ID              string            // Unique campaign ID
    Name            string            // Campaign name (for reporting)
    AdvertiserID    string            // Advertiser identifier
    Status          CampaignStatus    // active, paused, budget_capped
    DailyBudget     float64           // Daily budget cap in USD
    SpentToday      float64           // Amount spent today (updated on wins)
    BudgetResetTime time.Time         // Next reset time (midnight UTC)

    // Targeting Rules
    Targeting       TargetingRules    // Geo, device, site/app targeting

    // Bid Strategy
    BidStrategy     BidStrategy       // fixed_cpm, dynamic
    FixedCPM        float64           // Fixed bid price (if strategy = fixed_cpm)
    MaxCPM          float64           // Maximum bid price (if strategy = dynamic)

    // Creative Assignment
    CreativeIDs     []string          // Allowed creative IDs for this campaign

    // Metadata
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type CampaignStatus string
const (
    StatusActive       CampaignStatus = "active"
    StatusPaused       CampaignStatus = "paused"
    StatusBudgetCapped CampaignStatus = "budget_capped"
)

type BidStrategy string
const (
    StrategyFixedCPM BidStrategy = "fixed_cpm"
    StrategyDynamic  BidStrategy = "dynamic"
)

type TargetingRules struct {
    GeoTargeting    []string  // Country codes (e.g., ["US", "CA"])
    DeviceTypes     []string  // Device types: "mobile", "desktop", "tablet"
    SiteDomains     []string  // Whitelisted site domains (empty = all allowed)
    AppBundles      []string  // Whitelisted app bundles (empty = all allowed)
    MinViewability  float64   // Minimum viewability score (0-1)
}
```

**Validation Rules** (from FR-011):
- `DailyBudget` MUST be > 0
- `SpentToday` MUST be <= `DailyBudget`
- `Status` = `budget_capped` when `SpentToday` >= `DailyBudget`
- `FixedCPM` MUST be > 0 if `BidStrategy` = `fixed_cpm`

**Matching Logic** (from FR-004):
```go
func (c *Campaign) MatchesBidRequest(req *BidRequest) bool {
    if c.Status != StatusActive {
        return false // Skip paused/budget-capped campaigns
    }

    // Geo targeting
    if !c.matchesGeo(req.Device.Geo) {
        return false
    }

    // Device targeting
    if !c.matchesDevice(req.Device.DeviceType) {
        return false
    }

    // Site/App targeting
    if req.Site != nil && !c.matchesSite(req.Site.Domain) {
        return false
    }
    if req.App != nil && !c.matchesApp(req.App.Bundle) {
        return false
    }

    return true
}
```

**Budget Management** (from FR-011):
- Budget checked before bidding: `SpentToday + BidPrice <= DailyBudget`
- Budget deducted on win notification receipt
- Status changed to `budget_capped` when budget exhausted
- Reset at midnight UTC via background goroutine

**Storage**:
- **Hot Path**: In-memory RWMutex-protected map (reloaded every 60 seconds)
- **Cold Storage**: PostgreSQL `campaigns` table

**PostgreSQL Schema**:
```sql
CREATE TABLE campaigns (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    advertiser_id   TEXT NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('active', 'paused', 'budget_capped')),
    daily_budget    DECIMAL(10,2) NOT NULL CHECK (daily_budget > 0),
    spent_today     DECIMAL(10,2) NOT NULL DEFAULT 0,
    budget_reset_time TIMESTAMPTZ NOT NULL,
    targeting       JSONB NOT NULL, -- TargetingRules as JSON
    bid_strategy    TEXT NOT NULL,
    fixed_cpm       DECIMAL(10,2),
    max_cpm         DECIMAL(10,2),
    creative_ids    TEXT[] NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_campaigns_status ON campaigns(status);
CREATE INDEX idx_campaigns_advertiser ON campaigns(advertiser_id);
```

---

### Creative

Ad asset approved for serving.

**Purpose**: Banner creative markup and metadata

**Schema**:
```go
type Creative struct {
    ID                 string      // Unique creative ID
    CampaignID         string      // Parent campaign ID
    Name               string      // Creative name (for reporting)
    Width              int         // Width in pixels (e.g., 300)
    Height             int         // Height in pixels (e.g., 250)
    Format             CreativeFormat // banner (only format for this feature)
    Markup             string      // HTML markup for banner creative
    ClickThroughURL    string      // Landing page URL
    AdvertiserDomain   string      // Advertiser domain (e.g., "example.com")
    ApprovalStatus     ApprovalStatus // approved, pending, rejected
    ExchangeApprovals  map[string]bool // Exchange-specific approval (e.g., "google_adx": true)

    // Tracking
    ImpressionTrackers []string    // Impression tracking pixel URLs
    ClickTrackers      []string    // Click tracking URLs

    // Metadata
    CreatedAt          time.Time
    UpdatedAt          time.Time
}

type CreativeFormat string
const (
    FormatBanner CreativeFormat = "banner"
    // Future: FormatVideo, FormatNative
)

type ApprovalStatus string
const (
    ApprovalApproved ApprovalStatus = "approved"
    ApprovalPending  ApprovalStatus = "pending"
    ApprovalRejected ApprovalStatus = "rejected"
)
```

**Validation Rules** (from FR-015):
- `Width` and `Height` MUST match standard IAB dimensions (e.g., 300x250, 728x90, 160x600)
- `Markup` MUST be valid HTML (sanitized, no scripts for security)
- `ApprovalStatus` MUST be `approved` before serving
- `ExchangeApprovals["google_adx"]` MUST be true for Google Ad Exchange

**Dimension Matching**:
```go
func (c *Creative) MatchesImpression(imp *Impression) bool {
    if imp.Banner == nil {
        return false
    }

    // Check exact dimension match
    if imp.Banner.W != nil && imp.Banner.H != nil {
        return c.Width == *imp.Banner.W && c.Height == *imp.Banner.H
    }

    // Check format array
    for _, format := range imp.Banner.Format {
        if c.Width == format.W && c.Height == format.H {
            return true
        }
    }

    return false
}
```

**Storage**:
- **Hot Path**: In-memory map (loaded with campaigns)
- **Cold Storage**: PostgreSQL `creatives` table

**PostgreSQL Schema**:
```sql
CREATE TABLE creatives (
    id                  TEXT PRIMARY KEY,
    campaign_id         TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    width               INT NOT NULL,
    height              INT NOT NULL,
    format              TEXT NOT NULL DEFAULT 'banner',
    markup              TEXT NOT NULL,
    click_through_url   TEXT NOT NULL,
    advertiser_domain   TEXT NOT NULL,
    approval_status     TEXT NOT NULL CHECK (approval_status IN ('approved', 'pending', 'rejected')),
    exchange_approvals  JSONB NOT NULL DEFAULT '{}',
    impression_trackers TEXT[] NOT NULL DEFAULT '{}',
    click_trackers      TEXT[] NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_creatives_campaign ON creatives(campaign_id);
CREATE INDEX idx_creatives_dimensions ON creatives(width, height);
CREATE INDEX idx_creatives_approval ON creatives(approval_status);
```

---

### WinNotification

Post-auction notification confirming bid won the auction.

**Source**: OpenRTB 2.5 Win Notification (sent to NURL)

**Purpose**: Track wins, deduct budget, calculate metrics

**Schema**:
```go
type WinNotification struct {
    ID             string    // Unique win notification ID
    BidID          string    // Matches Bid.ID from bid response
    ImpressionID   string    // Matches Impression.ID from bid request
    AuctionID      string    // Matches BidRequest.ID
    CampaignID     string    // Campaign that won
    CreativeID     string    // Creative that was served
    WinPrice       float64   // Final settlement price (CPM in USD)
    BidPrice       float64   // Original bid price (may differ from win price)
    Currency       string    // Currency code (default: "USD")

    // Attribution
    AdvertiserID   string    // Advertiser identifier
    SiteDomain     string    // Site where ad was served (if site impression)
    AppBundle      string    // App where ad was served (if app impression)
    DeviceType     string    // Device type (mobile, desktop, tablet)
    GeoCountry     string    // Country code (e.g., "US")

    // Timestamps
    ReceivedAt     time.Time // When win notification received by bidder
    AuctionTime    time.Time // When original auction occurred (from bid request)
}
```

**Processing Rules** (from FR-010):
1. Match `BidID` to original bid (lookup in recent bids cache - 5 minute TTL)
2. Deduct `WinPrice` from `Campaign.SpentToday`
3. Update campaign status to `budget_capped` if budget exhausted
4. Emit metrics event for win tracking

**Reconciliation** (from User Story 3, Acceptance Scenario 4):
- Unmatched wins (no corresponding bid found) flagged after 5 minute window
- Logged for investigation (possible duplicate notification or cache miss)

**Storage**:
- **Immediate**: Write to buffered channel (off critical path)
- **Persistent**: InfluxDB for time-series metrics analysis

**InfluxDB Schema**:
```
Measurement: wins
Tags: campaign_id, creative_id, advertiser_id, device_type, geo_country
Fields: win_price, bid_price, latency_ms (auction_time → received_at)
Timestamp: received_at
```

---

### BidMetrics

Aggregated metrics for monitoring and reporting.

**Purpose**: Support User Story 3 - metrics tracking and performance optimization

**Schema** (in-memory aggregation, flushed to InfluxDB):
```go
type BidMetrics struct {
    // Time Window
    WindowStart time.Time
    WindowEnd   time.Time

    // Campaign Aggregation
    CampaignID  string

    // Request Metrics
    TotalRequests      int64   // Total bid requests received
    ValidRequests      int64   // Requests passing validation
    InvalidRequests    int64   // Requests failing validation

    // Bid Metrics
    TotalBids          int64   // Total bids submitted
    NoBids             int64   // No-bid responses

    // Win Metrics
    TotalWins          int64   // Total wins received
    TotalSpend         float64 // Total spend (sum of win prices)
    AverageCPM         float64 // Average winning CPM

    // Performance Metrics
    WinRate            float64 // Wins / Bids (percentage)
    BidRate            float64 // Bids / Requests (percentage)
    AverageLatencyMs   float64 // Average bid processing latency
    P95LatencyMs       float64 // p95 bid processing latency
    P99LatencyMs       float64 // p99 bid processing latency

    // Error Metrics
    TimeoutErrors      int64   // Requests exceeding timeout
    ValidationErrors   int64   // OpenRTB validation failures
}
```

**Calculation Rules** (from User Story 3, Acceptance Scenario 2):
- `WinRate = (TotalWins / TotalBids) * 100`
- `BidRate = (TotalBids / ValidRequests) * 100`
- `AverageCPM = TotalSpend / TotalWins`

**Latency Tracking**:
- Measure from request receipt to response sent
- Use histogram for percentile calculations (p95, p99)
- Prometheus histogram buckets: [10ms, 25ms, 50ms, 75ms, 100ms, 150ms, 200ms]

**Storage**: InfluxDB with 1-minute aggregation buckets

---

## Entity Relationships

```
BidRequest (1) ──contains──> (N) Impression
BidRequest (1) ──generates──> (0..1) BidResponse
BidResponse (1) ──contains──> (N) Bid
Bid (1) ──references──> (1) Impression
Bid (1) ──triggers──> (0..1) WinNotification

Campaign (1) ──owns──> (N) Creative
Campaign (1) ──matches──> (N) BidRequest [via targeting]
Creative (1) ──selected_for──> (N) Bid
Creative (1) ──matches──> (N) Impression [via dimensions]

WinNotification (1) ──updates──> (1) Campaign [budget deduction]
WinNotification (N) ──aggregated_into──> (1) BidMetrics
```

---

## Data Flow

### Request Processing (User Story 1, P1):
1. Receive `BidRequest` from Google Ad Exchange (HTTP POST)
2. Unmarshal JSON into `openrtb2.BidRequest` struct
3. Validate required fields (ID, Imp array, Site/App, Device)
4. Extract targeting criteria from `Impression.Banner` and `Device`

### Bid Generation (User Story 2, P2):
1. Query in-memory campaign store for active campaigns
2. Filter campaigns by targeting match (`Campaign.MatchesBidRequest`)
3. For each matching campaign:
   - Check budget: `SpentToday + FixedCPM <= DailyBudget`
   - Find creative matching impression dimensions
   - Calculate bid price from `Campaign.BidStrategy`
4. Select highest priority campaign (highest bid price)
5. Construct `BidResponse` with selected `Bid`
6. Marshal to JSON and return to Google Ad Exchange

### Win Processing (User Story 3, P3):
1. Receive `WinNotification` (HTTP POST to NURL callback)
2. Match `BidID` to original bid (in-memory cache lookup)
3. Write to buffered channel (async processing):
   - Update `Campaign.SpentToday` in PostgreSQL
   - Update campaign status if budget exhausted
   - Write win event to InfluxDB
   - Emit Prometheus metrics

---

## Storage Strategy Summary

| Entity | Hot Path (In-Memory) | Cold Storage (PostgreSQL) | Metrics (InfluxDB) |
|--------|---------------------|---------------------------|-------------------|
| **BidRequest** | Ephemeral (not stored) | - | - |
| **BidResponse** | Ephemeral (logged only) | - | Latency metrics |
| **Campaign** | RWMutex map (60s reload) | `campaigns` table | - |
| **Creative** | RWMutex map (with campaigns) | `creatives` table | - |
| **WinNotification** | 5-min cache for reconciliation | - | `wins` measurement |
| **BidMetrics** | 1-min aggregation window | - | `metrics` measurement |

**Hot Path Priority**: Campaign and Creative data MUST be in-memory to meet 100ms p95 latency requirement.

---

## Next Steps

With data model defined, proceed to:
1. **API Contracts**: Generate OpenRTB request/response schemas in /contracts/
2. **Quickstart**: Document local development setup with PostgreSQL and InfluxDB
