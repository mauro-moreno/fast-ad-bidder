# Quickstart Guide: OpenRTB Bidder Development

**Feature**: OpenRTB Bidder (Google Services Integration)
**Date**: 2025-12-27
**Purpose**: Local development setup and testing guide

## Prerequisites

### Required Tools
- **Go**: 1.21 or later
- **PostgreSQL**: 14 or later (campaign storage)
- **InfluxDB**: 2.x (metrics storage)
- **Docker**: 20.10+ and Docker Compose (for local services)
- **Git**: 2.30+

### Optional Tools
- **k6**: Load testing
- **Postman/curl**: API testing
- **Prometheus**: Metrics visualization (included in Docker Compose)
- **Grafana**: Dashboards (included in Docker Compose)

---

## Quick Start (5 Minutes)

### 1. Clone Repository

```bash
git clone <repository-url>
cd fast-ad-bidder
git checkout 001-openrtb-bidder
```

### 2. Start Dependencies

```bash
# Start PostgreSQL, InfluxDB, Prometheus, Grafana
docker-compose up -d

# Verify services are running
docker-compose ps
```

Expected output:
```
NAME                COMMAND                  STATUS              PORTS
postgres            "docker-entrypoint.s…"   Up                  0.0.0.0:5432->5432/tcp
influxdb            "/entrypoint.sh infl…"   Up                  0.0.0.0:8086->8086/tcp
prometheus          "/bin/prometheus --c…"   Up                  0.0.0.0:9090->9090/tcp
grafana             "/run.sh"                Up                  0.0.0.0:3000->3000/tcp
```

### 3. Initialize Database

```bash
# Create campaigns table
psql postgres://bidder:password@localhost:5432/bidder -f config/schema.sql

# Load sample campaigns
psql postgres://bidder:password@localhost:5432/bidder -f config/sample_data.sql
```

### 4. Install Go Dependencies

```bash
go mod download
```

### 5. Run Bidder Service

```bash
# Start bidder (defaults to port 8080)
go run cmd/bidder/main.go

# Or with hot reload (install air first: go install github.com/cosmtrek/air@latest)
air
```

### 6. Verify Service is Running

```bash
# Health check
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","version":"dev","campaigns_loaded":5}
```

---

## Development Workflow

### Project Structure

```
fast-ad-bidder/
├── cmd/
│   └── bidder/              # Main application entry point
│       └── main.go
├── src/
│   ├── models/              # Data models (Campaign, Creative, BidRequest, etc.)
│   ├── services/            # Business logic
│   │   ├── validator/       # OpenRTB request validation
│   │   ├── matcher/         # Campaign targeting matcher
│   │   ├── pricer/          # Bid price calculator
│   │   └── tracker/         # Win notification tracker
│   ├── api/                 # HTTP handlers
│   ├── middleware/          # mTLS auth, logging, metrics
│   └── lib/                 # Shared utilities
├── tests/
│   ├── contract/            # OpenRTB contract tests
│   ├── integration/         # End-to-end tests
│   └── unit/                # Unit tests
├── config/
│   ├── schema.sql           # PostgreSQL schema
│   ├── sample_data.sql      # Sample campaigns/creatives
│   └── campaigns/           # Campaign JSON configs
├── docker-compose.yml       # Local development services
├── go.mod
└── go.sum
```

### Configuration

**Environment Variables** (`.env` file):
```bash
# Server
PORT=8080
ENABLE_TLS=false  # Set to true for mTLS in production

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_NAME=bidder
DB_USER=bidder
DB_PASSWORD=password

# InfluxDB
INFLUX_URL=http://localhost:8086
INFLUX_TOKEN=dev-token
INFLUX_ORG=fast-ad-bidder
INFLUX_BUCKET=metrics

# Campaign Reload
CAMPAIGN_RELOAD_INTERVAL=60s  # How often to reload campaigns from PostgreSQL

# Metrics
METRICS_ENABLED=true

# Logging
LOG_LEVEL=debug  # debug, info, warn, error
LOG_FORMAT=json  # json or console
```

---

## Testing

### Run Unit Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Contract Tests

```bash
# Install Pact Go CLI (first time only)
go install github.com/pact-foundation/pact-go/v2@latest

# Run contract tests
go test ./tests/contract/... -v
```

### Run Integration Tests

```bash
# Start test dependencies
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test ./tests/integration/... -v

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### Load Testing with k6

```bash
# Install k6 (macOS)
brew install k6

# Install k6 (Linux)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Run load test
k6 run tests/load/bid_endpoint.js

# With results output to InfluxDB
k6 run --out influxdb=http://localhost:8086/k6 tests/load/bid_endpoint.js
```

**Example k6 Test** (`tests/load/bid_endpoint.js`):
```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 100 },   // Ramp up to 100 RPS
    { duration: '1m', target: 500 },    // Ramp up to 500 RPS
    { duration: '2m', target: 1000 },   // Ramp up to 1000 RPS
    { duration: '1m', target: 1000 },   // Hold at 1000 RPS
    { duration: '30s', target: 0 },     // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<100', 'p(99)<120'], // p95 < 100ms, p99 < 120ms
    http_req_failed: ['rate<0.01'],                // Error rate < 1%
  },
};

const bidRequest = {
  id: '${__VU}-${__ITER}',
  imp: [
    {
      id: 'imp-1',
      banner: { w: 300, h: 250 },
      bidfloor: 0.5,
    },
  ],
  site: {
    domain: 'example.com',
    page: 'https://example.com/article',
  },
  device: {
    ua: 'Mozilla/5.0...',
    ip: '192.0.2.1',
    devicetype: 2,
    geo: { country: 'USA' },
  },
  tmax: 120,
};

export default function () {
  const res = http.post(
    'http://localhost:8080/bid',
    JSON.stringify(bidRequest),
    {
      headers: { 'Content-Type': 'application/json' },
    }
  );

  check(res, {
    'status is 200': (r) => r.status === 200,
    'has bid response': (r) => r.json('id') !== undefined,
    'latency < 100ms': (r) => r.timings.duration < 100,
  });
}
```

---

## Sample API Requests

### 1. Send Bid Request

```bash
curl -X POST http://localhost:8080/bid \
  -H "Content-Type: application/json" \
  -d '{
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
      "domain": "example.com",
      "page": "https://example.com/article"
    },
    "device": {
      "ua": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
      "ip": "192.0.2.1",
      "devicetype": 2,
      "geo": {
        "country": "USA"
      }
    },
    "tmax": 120
  }'
```

**Expected Response** (if matching campaign exists):
```json
{
  "id": "auction-123",
  "seatbid": [
    {
      "bid": [
        {
          "id": "bid-abc123",
          "impid": "imp-1",
          "price": 2.50,
          "adid": "creative-1",
          "adomain": ["advertiser.com"],
          "crid": "creative-1",
          "w": 300,
          "h": 250,
          "nurl": "http://localhost:8080/win?bid=bid-abc123&price=${AUCTION_PRICE}"
        }
      ]
    }
  ],
  "cur": "USD"
}
```

### 2. Trigger Win Notification

```bash
# Use bid ID from previous response
curl "http://localhost:8080/win?bid=bid-abc123&price=2500000&currency=USD"
```

**Expected Response**:
```json
{
  "status": "ok"
}
```

### 3. Check Health

```bash
curl http://localhost:8080/health
```

### 4. View Metrics

```bash
curl http://localhost:8080/metrics
```

---

## Database Management

### View Campaigns

```bash
psql postgres://bidder:password@localhost:5432/bidder

SELECT id, name, status, daily_budget, spent_today
FROM campaigns
WHERE status = 'active';
```

### Add New Campaign

```sql
INSERT INTO campaigns (
  id, name, advertiser_id, status, daily_budget, spent_today,
  budget_reset_time, targeting, bid_strategy, fixed_cpm, creative_ids
) VALUES (
  'campaign-123',
  'Test Campaign',
  'advertiser-1',
  'active',
  100.00,
  0.00,
  NOW() + INTERVAL '1 day',
  '{"GeoTargeting": ["US"], "DeviceTypes": ["mobile", "desktop"]}',
  'fixed_cpm',
  2.50,
  ARRAY['creative-1']
);
```

### Add Creative

```sql
INSERT INTO creatives (
  id, campaign_id, name, width, height, format, markup,
  click_through_url, advertiser_domain, approval_status, exchange_approvals
) VALUES (
  'creative-1',
  'campaign-123',
  'Banner 300x250',
  300,
  250,
  'banner',
  '<div>Ad Content Here</div>',
  'https://advertiser.com/landing',
  'advertiser.com',
  'approved',
  '{"google_adx": true}'
);
```

### Check Budget Status

```sql
SELECT
  id,
  name,
  daily_budget,
  spent_today,
  (daily_budget - spent_today) AS remaining,
  status
FROM campaigns
ORDER BY spent_today DESC;
```

---

## Monitoring

### Prometheus Metrics

Access Prometheus: http://localhost:9090

**Useful Queries**:

Request Rate:
```promql
rate(bidder_requests_total[1m])
```

Latency (p95):
```promql
histogram_quantile(0.95, rate(bidder_latency_seconds_bucket[1m]))
```

Win Rate:
```promql
rate(bidder_wins_total[5m]) / rate(bidder_bids_total{result="bid"}[5m])
```

Error Rate:
```promql
rate(bidder_errors_total[1m])
```

### Grafana Dashboards

Access Grafana: http://localhost:3000 (admin/admin)

**Pre-configured Dashboards**:
1. **Bidder Overview**: Request rate, latency, win rate
2. **Campaign Performance**: Budget tracking, spend rate per campaign
3. **System Health**: CPU, memory, goroutines

Import dashboard JSON from: `config/grafana/dashboards/bidder.json`

### InfluxDB Metrics

Access InfluxDB UI: http://localhost:8086

**Useful Queries** (Flux):

Win rate over time:
```flux
from(bucket: "metrics")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "wins")
  |> aggregateWindow(every: 1m, fn: count)
```

Campaign spend:
```flux
from(bucket: "metrics")
  |> range(start: -24h)
  |> filter(fn: (r) => r._measurement == "wins")
  |> group(columns: ["campaign_id"])
  |> sum(column: "win_price")
```

---

## Troubleshooting

### Service Won't Start

**Error**: `failed to connect to PostgreSQL`
```bash
# Check PostgreSQL is running
docker-compose ps postgres

# Check connection
psql postgres://bidder:password@localhost:5432/bidder -c "SELECT 1"

# Restart PostgreSQL
docker-compose restart postgres
```

**Error**: `campaigns table does not exist`
```bash
# Run schema migration
psql postgres://bidder:password@localhost:5432/bidder -f config/schema.sql
```

### No Bids Returned

**Symptom**: Bid requests return empty `seatbid` array

**Debugging**:
1. Check campaigns are loaded:
   ```bash
   curl http://localhost:8080/health | jq '.campaigns_loaded'
   ```

2. Check campaign status:
   ```sql
   SELECT id, name, status FROM campaigns;
   ```

3. Enable debug logging:
   ```bash
   LOG_LEVEL=debug go run cmd/bidder/main.go
   ```

4. Check targeting match:
   - Verify request geo matches campaign targeting
   - Verify device type matches campaign targeting
   - Verify creative dimensions match impression

### High Latency

**Symptom**: p95 latency > 100ms

**Debugging**:
1. Check database query performance:
   ```sql
   EXPLAIN ANALYZE SELECT * FROM campaigns WHERE status = 'active';
   ```

2. Verify campaigns are in memory (not hitting database on hot path):
   - Check logs for "Campaign reload completed" messages
   - Verify `CAMPAIGN_RELOAD_INTERVAL` is set (default 60s)

3. Profile with pprof:
   ```bash
   # Enable pprof (add to main.go)
   import _ "net/http/pprof"

   # View CPU profile
   go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30

   # View memory allocations
   go tool pprof http://localhost:8080/debug/pprof/heap
   ```

### Win Notifications Not Tracked

**Symptom**: 404 errors on /win endpoint

**Cause**: Bid cache expired (TTL: 5 minutes)

**Solution**:
- Increase bid cache TTL if win notifications are delayed
- Check win notification timing in logs
- Verify `nurl` field in bid response contains correct URL

---

## Production Deployment Checklist

Before deploying to production:

- [ ] Enable mTLS authentication (`ENABLE_TLS=true`)
- [ ] Configure production TLS certificates
- [ ] Set `LOG_LEVEL=info` (not debug)
- [ ] Configure PostgreSQL connection pooling
- [ ] Set up InfluxDB retention policies (30 days for metrics)
- [ ] Configure Prometheus alerts for:
  - [ ] p95 latency > 100ms
  - [ ] Error rate > 1%
  - [ ] Campaign budget exhaustion
  - [ ] Service unhealthy
- [ ] Load test at 2x expected peak traffic (2000 QPS)
- [ ] Configure horizontal pod autoscaling (Kubernetes)
- [ ] Set up log aggregation (e.g., ELK, Loki)
- [ ] Document runbook for common incidents

---

## Resources

- **OpenRTB 2.5 Spec**: https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf
- **Google Ad Exchange RTB Guide**: https://developers.google.com/authorized-buyers/rtb/openrtb-guide
- **Prebid OpenRTB Library**: https://github.com/prebid/openrtb
- **Echo Framework**: https://echo.labstack.com/
- **Pact Contract Testing**: https://docs.pact.io/
- **k6 Load Testing**: https://k6.io/docs/

---

## Next Steps

After local development setup:
1. Review [data-model.md](data-model.md) for entity schemas
2. Review [contracts/README.md](contracts/README.md) for API design
3. Run `/speckit.tasks` to generate implementation tasks
4. Start with P1 user story (Request Validation) implementation
