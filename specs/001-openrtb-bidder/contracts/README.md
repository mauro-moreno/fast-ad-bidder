# API Contracts: OpenRTB Bidder

**Feature**: OpenRTB Bidder (Google Services Integration)
**Date**: 2025-12-27
**Purpose**: Define HTTP API contracts for bidder service

## Contract Files

- **openapi.yaml**: OpenAPI 3.0 specification for all bidder endpoints

## Endpoints Overview

### 1. POST /bid (Bidding Endpoint)

**Purpose**: Receive OpenRTB 2.5 bid requests from Google Ad Exchange

**User Stories**: P1 (Request Validation), P2 (Bid Generation)

**Request**: OpenRTB 2.5 BidRequest JSON
**Response**: OpenRTB 2.5 BidResponse JSON (or empty for no-bid)

**Performance SLA**: Must respond within 100ms at p95 latency

**Authentication**: Requires mutual TLS (mTLS) with valid client certificate

**Validation Rules** (from FR-002):
- Request ID must be present
- At least one impression required
- Either Site or App must be present (not both)
- Device information required

**Example Request**:
```json
{
  "id": "auction-123",
  "imp": [
    {
      "id": "imp-1",
      "banner": {
        "w": 300,
        "h": 250
      },
      "bidfloor": 0.50
    }
  ],
  "site": {
    "domain": "example.com"
  },
  "device": {
    "ip": "192.0.2.1",
    "devicetype": 2,
    "geo": {
      "country": "USA"
    }
  }
}
```

**Example Response (Successful Bid)**:
```json
{
  "id": "auction-123",
  "seatbid": [
    {
      "bid": [
        {
          "id": "bid-456",
          "impid": "imp-1",
          "price": 2.50,
          "adid": "creative-789",
          "adomain": ["advertiser.com"],
          "crid": "creative-789",
          "w": 300,
          "h": 250,
          "nurl": "https://bidder.example.com/win?bid=bid-456&price=${AUCTION_PRICE}"
        }
      ]
    }
  ],
  "cur": "USD"
}
```

**Example Response (No Bid)**:
```json
{
  "id": "auction-123",
  "seatbid": [],
  "nbr": 2,
  "cur": "USD"
}
```

**No-Bid Reason Codes** (NBR):
- `0`: Unknown error
- `1`: Technical error
- `2`: Invalid request
- `3`: Known web spider
- `4`: Suspected non-human traffic
- `100`: Timeout
- `200`: No matching campaigns
- `201`: Budget exhausted

---

### 2. GET /win (Win Notification Endpoint)

**Purpose**: Receive win notifications from Google Ad Exchange

**User Story**: P3 (Metrics Tracking)

**Parameters**:
- `bid` (required): Bid ID from original bid response
- `price` (required): Final settlement price in micros (CPM * 1,000,000)
- `currency` (optional): Currency code (default: USD)

**Processing**:
1. Look up original bid by ID (in-memory cache, 5-minute TTL)
2. Convert price from micros to CPM: `price / 1,000,000`
3. Queue win notification for async processing:
   - Update campaign budget in PostgreSQL
   - Write win event to InfluxDB
   - Emit Prometheus metrics
4. Return 200 OK immediately

**Example Request**:
```
GET /win?bid=bid-456&price=2500000&currency=USD
```

**Example Response**:
```json
{
  "status": "ok"
}
```

**Error Scenarios**:
- `404 Not Found`: Bid ID not found (orphaned win or cache expiration)
- `500 Internal Server Error`: Database or processing failure

**Reconciliation**:
- Unmatched wins (404 responses) are logged for investigation
- Bids older than 5 minutes are removed from cache to prevent memory growth

---

### 3. GET /health (Health Check Endpoint)

**Purpose**: Service health status for load balancers and monitoring

**Constitution Requirement**: All services must expose health check endpoints

**Authentication**: None (public endpoint)

**Health Criteria**:
- Service is running (process alive)
- Campaign store is initialized (campaigns loaded)
- Recent successful database connection (within last 60 seconds)

**Response (Healthy)**:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 3600,
  "campaigns_loaded": 42,
  "last_reload": "2025-12-27T12:00:00Z"
}
```

**Response (Unhealthy)**:
```json
{
  "status": "unhealthy",
  "version": "1.0.0",
  "error": "Campaign store not initialized"
}
```

**HTTP Status Codes**:
- `200 OK`: Service is healthy
- `503 Service Unavailable`: Service is unhealthy

**Use Cases**:
- Load balancer health checks (poll every 10 seconds)
- Kubernetes liveness/readiness probes
- Deployment validation

---

### 4. GET /metrics (Prometheus Metrics Endpoint)

**Purpose**: Expose Prometheus metrics for monitoring

**Constitution Requirement**: Services must expose metrics for latency, throughput, error rates

**Authentication**: None (internal network only, not exposed publicly)

**Metrics Exposed**:

**Request Metrics**:
- `bidder_requests_total{status="valid|invalid"}` - Counter of bid requests by validation status
- `bidder_bids_total{result="bid|nobid"}` - Counter of bid responses
- `bidder_wins_total` - Counter of win notifications received

**Latency Metrics** (Histogram):
- `bidder_latency_seconds` - Bid processing latency
  - Buckets: [0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.2]
  - Labels: `endpoint="/bid"`

**Error Metrics**:
- `bidder_errors_total{type="validation|timeout|internal"}` - Counter of errors by type

**Campaign Metrics**:
- `bidder_campaigns_active` - Gauge of active campaigns loaded
- `bidder_campaign_budget_remaining{campaign_id}` - Gauge of remaining daily budget

**Example Output**:
```
# HELP bidder_requests_total Total bid requests received
# TYPE bidder_requests_total counter
bidder_requests_total{status="valid"} 10000
bidder_requests_total{status="invalid"} 42

# HELP bidder_latency_seconds Bid processing latency
# TYPE bidder_latency_seconds histogram
bidder_latency_seconds_bucket{endpoint="/bid",le="0.01"} 1000
bidder_latency_seconds_bucket{endpoint="/bid",le="0.05"} 8000
bidder_latency_seconds_bucket{endpoint="/bid",le="0.1"} 9500
bidder_latency_seconds_bucket{endpoint="/bid",le="+Inf"} 10000
bidder_latency_seconds_sum{endpoint="/bid"} 450.5
bidder_latency_seconds_count{endpoint="/bid"} 10000
```

**Prometheus Configuration**:
```yaml
scrape_configs:
  - job_name: 'bidder'
    scrape_interval: 15s
    static_configs:
      - targets: ['bidder:8080']
```

---

## OpenRTB 2.5 Compliance

This API implements the following OpenRTB 2.5 specification sections:

### Required Objects
- ✅ **BidRequest**: Core request object (Section 3.2.1)
- ✅ **Impression**: Impression object (Section 3.2.2)
- ✅ **Banner**: Banner object (Section 3.2.3) - only format supported
- ✅ **Site**: Website object (Section 3.2.6)
- ✅ **App**: Mobile app object (Section 3.2.7)
- ✅ **Device**: Device object (Section 3.2.11)
- ✅ **Geo**: Geographic location (Section 3.2.12)
- ✅ **User**: User object (Section 3.2.13)
- ✅ **BidResponse**: Core response object (Section 4.2.1)
- ✅ **SeatBid**: Seat bid object (Section 4.2.2)
- ✅ **Bid**: Bid object (Section 4.2.3)

### Not Implemented (Out of Scope)
- ❌ **Video**: Video object (Section 3.2.4) - banner-only scope
- ❌ **Audio**: Audio object (Section 3.2.5) - banner-only scope
- ❌ **Native**: Native object (Section 3.2.9) - banner-only scope
- ❌ **PMP**: Private marketplace (Section 3.2.17) - future enhancement

### No-Bid Reason Codes (NBR)
Conforms to OpenRTB 2.5 Section 5.19 (No-Bid Reason Codes):
- Code 2: Invalid request
- Code 100: Timeout
- Code 200: No matching campaigns (custom extension)
- Code 201: Budget exhausted (custom extension)

---

## Security

### Mutual TLS (mTLS) Authentication

**Requirement** (from FR-013): All /bid and /win endpoints require mutual TLS

**Implementation**:
- Client certificate verification enabled
- Minimum TLS version: 1.3
- Certificate validation against trusted CA bundle
- Certificate expiration monitoring

**Certificate Management**:
- Short-lived certificates (30-90 days)
- Automated rotation via cert-manager (Kubernetes) or AWS ACM
- Certificate expiration alerts (Prometheus alert: 7 days before expiry)

**Configuration Example** (Go):
```go
tlsConfig := &tls.Config{
    ClientCAs:  caCertPool,
    ClientAuth: tls.RequireAndVerifyClientCert,
    MinVersion: tls.VersionTLS13,
}
server := &http.Server{
    Addr:      ":8443",
    TLSConfig: tlsConfig,
    Handler:   router,
}
```

**Allowed Client Certificates**:
- Google Ad Exchange production certificates
- Google Ad Exchange sandbox certificates (for testing)

---

## Testing

### Contract Testing

**Framework**: Pact Go v2

**Test Coverage**:
1. **Valid Bid Request** → 200 with valid BidResponse
2. **Invalid Bid Request** (missing imp) → 400 with error
3. **No Matching Campaigns** → 200 with empty seatbid
4. **Win Notification** → 200 with status ok
5. **Orphaned Win** → 404 with error

**Pact Provider Verification**:
```bash
# Verify bidder service against Pact contracts
pact-go verify \
  --provider "openrtb-bidder" \
  --pact-urls ./contracts/pacts \
  --provider-base-url https://localhost:8443
```

### Load Testing

**Framework**: k6

**Performance Validation**:
- Ramp to 1000 QPS over 2 minutes
- Maintain 1000 QPS for 5 minutes
- Verify p95 latency < 100ms
- Verify p99 latency < 120ms

**k6 Script**:
```javascript
import http from 'k6/http';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '2m', target: 1000 },
    { duration: '5m', target: 1000 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<100', 'p(99)<120'],
  },
};

export default function () {
  const bidRequest = {
    id: `auction-${__VU}-${__ITER}`,
    imp: [{ id: 'imp-1', banner: { w: 300, h: 250 } }],
    site: { domain: 'example.com' },
    device: { ip: '192.0.2.1', devicetype: 2 },
  };

  const res = http.post('https://bidder.example.com/bid', JSON.stringify(bidRequest), {
    headers: { 'Content-Type': 'application/json' },
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
    'response is valid OpenRTB': (r) => r.json('id') !== undefined,
  });
}
```

---

## OpenAPI Validation

**Tools**:
- **Swagger UI**: Interactive API documentation
- **Redocly**: API reference documentation
- **Spectral**: OpenAPI linting for best practices

**Validation Commands**:
```bash
# Validate OpenAPI spec
npx @redocly/cli lint contracts/openapi.yaml

# Generate interactive docs
npx @redocly/cli preview-docs contracts/openapi.yaml

# Validate requests/responses against schema
npm install -g openapi-enforcer
openapi-enforcer validate contracts/openapi.yaml
```

---

## References

- OpenRTB 2.5 Specification: https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf
- Google Ad Exchange RTB Guide: https://developers.google.com/authorized-buyers/rtb/openrtb-guide
- Prebid OpenRTB Library: https://github.com/prebid/openrtb
- OpenAPI 3.0 Specification: https://spec.openapis.org/oas/v3.0.3

---

## Usage Examples

### Testing with curl

#### 1. Send Bid Request

```bash
curl -X POST http://localhost:8080/bid \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-auction-001",
    "imp": [{
      "id": "imp-1",
      "banner": {
        "w": 300,
        "h": 250
      },
      "bidfloor": 1.0
    }],
    "site": {
      "domain": "example.com",
      "page": "https://example.com/news/article"
    },
    "device": {
      "ip": "192.0.2.1",
      "devicetype": 2,
      "geo": {
        "country": "USA",
        "region": "CA"
      }
    }
  }'
```

#### 2. Send Win Notification

```bash
# Extract bid ID from response above, then:
curl -X GET "http://localhost:8080/win?bid=<BID_ID>&price=2500000&currency=USD"
```

#### 3. Health Check

```bash
curl http://localhost:8080/health
```

#### 4. Prometheus Metrics

```bash
curl http://localhost:8080/metrics
```

### Testing with mTLS (Production)

When mTLS is enabled, include client certificates:

```bash
curl -X POST https://localhost:8080/bid \
  --cert config/certs/client.crt \
  --key config/certs/client.key \
  --cacert config/certs/ca.crt \
  -H "Content-Type: application/json" \
  -d @tests/fixtures/bid_requests.json
```

### Python Example

```python
import requests
import json

# Bid request
bid_request = {
    "id": "python-test-001",
    "imp": [{
        "id": "imp-1",
        "banner": {"w": 728, "h": 90},
        "bidfloor": 1.5
    }],
    "site": {
        "domain": "news.example.com",
        "page": "https://news.example.com/article/123"
    },
    "device": {
        "ip": "192.0.2.10",
        "devicetype": 2,
        "geo": {"country": "USA", "region": "NY"}
    }
}

# Send bid request
response = requests.post(
    "http://localhost:8080/bid",
    json=bid_request,
    headers={"Content-Type": "application/json"}
)

print(f"Status: {response.status_code}")
print(f"Response: {json.dumps(response.json(), indent=2)}")

# Extract bid ID if successful
if response.status_code == 200:
    bid_response = response.json()
    if bid_response.get("seatbid"):
        bid_id = bid_response["seatbid"][0]["bid"][0]["id"]
        print(f"\nBid ID: {bid_id}")
        
        # Send win notification
        win_response = requests.get(
            f"http://localhost:8080/win?bid={bid_id}&price=2500000&currency=USD"
        )
        print(f"Win Status: {win_response.status_code}")
```

### Go Example

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

type BidRequest struct {
    ID     string       `json:"id"`
    Imp    []Impression `json:"imp"`
    Site   *Site        `json:"site,omitempty"`
    Device *Device      `json:"device"`
}

type Impression struct {
    ID       string  `json:"id"`
    Banner   *Banner `json:"banner,omitempty"`
    BidFloor float64 `json:"bidfloor"`
}

type Banner struct {
    W int `json:"w"`
    H int `json:"h"`
}

type Site struct {
    Domain string `json:"domain"`
    Page   string `json:"page"`
}

type Device struct {
    IP         string `json:"ip"`
    DeviceType int    `json:"devicetype"`
    Geo        *Geo   `json:"geo,omitempty"`
}

type Geo struct {
    Country string `json:"country"`
    Region  string `json:"region,omitempty"`
}

func main() {
    bidReq := BidRequest{
        ID: "go-test-001",
        Imp: []Impression{{
            ID:       "imp-1",
            Banner:   &Banner{W: 300, H: 250},
            BidFloor: 1.0,
        }},
        Site: &Site{
            Domain: "example.com",
            Page:   "https://example.com/page",
        },
        Device: &Device{
            IP:         "192.0.2.1",
            DeviceType: 2,
            Geo:        &Geo{Country: "USA"},
        },
    }

    body, _ := json.Marshal(bidReq)
    
    resp, err := http.Post(
        "http://localhost:8080/bid",
        "application/json",
        bytes.NewBuffer(body),
    )
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    respBody, _ := io.ReadAll(resp.Body)
    fmt.Printf("Status: %d\n", resp.StatusCode)
    fmt.Printf("Response: %s\n", string(respBody))
}
```

### Load Testing

```bash
# Quick smoke test
k6 run tests/load/bid_endpoint.js

# Full 1000 QPS validation
k6 run --duration 5m tests/load/bid_latency_test.js
```

---

## Contract Testing

The service includes Pact contract tests to ensure OpenRTB 2.5 compliance:

```bash
# Run contract tests
go test -v ./tests/contract/...

# Generate Pact files for consumer contract testing
go test -v ./tests/contract/bid_request_validation_test.go
```

## Monitoring

After sending requests, check metrics:

```bash
# View all metrics
curl http://localhost:8080/metrics

# Filter specific metrics
curl http://localhost:8080/metrics | grep http_requests_total
curl http://localhost:8080/metrics | grep wins_
curl http://localhost:8080/metrics | grep campaigns_budget
```

Key metrics to monitor:
- `http_request_duration_seconds` - Request latency histogram
- `bids_total{type="bid"}` - Successful bids counter
- `bids_total{type="nobid"}` - No-bid responses counter
- `wins_received_total` - Win notifications received
- `wins_orphaned_total` - Orphaned wins (not in cache)
- `active_campaigns_total` - Active campaigns with budget
- `campaigns_budget_remaining` - Remaining budget across campaigns

