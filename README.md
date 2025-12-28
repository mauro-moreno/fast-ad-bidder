# Fast Ad Bidder - OpenRTB 2.5 Bidder Service

A high-performance OpenRTB 2.5 bidder service for Google Ad Exchange integration, written in Go. Validates bid requests, matches campaigns with targeting rules, generates bid responses within 100ms p95 latency, and tracks win notifications for budget management.

## Features

- ✅ **OpenRTB 2.5 Compliance** - Full protocol compliance with Google Ad Exchange
- ✅ **Sub-100ms Latency** - p95 < 100ms, p99 < 120ms response times
- ✅ **mTLS Authentication** - Mutual TLS with bidirectional certificate validation
- ✅ **Campaign Targeting** - Geo, device, domain, and creative dimension matching
- ✅ **Budget Management** - Real-time spend tracking with daily budget caps and midnight UTC reset
- ✅ **Win Notification Processing** - Async win tracking with worker pools and 5-minute bid cache TTL
- ✅ **Production Observability** - Structured logging (zap), Prometheus metrics, InfluxDB integration
- ✅ **1000 QPS Capacity** - Handles 1000+ requests/second with horizontal scaling

## Table of Contents

- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Configuration](#configuration)
- [Performance & Monitoring](#performance--monitoring)
- [Development](#development)
- [Deployment](#deployment)
- [Troubleshooting](#troubleshooting)
- [API Examples](#api-examples)

## Quick Reference

```bash
# Setup and run locally
make services-up                              # Start Docker services
docker-compose exec -T postgres psql -U bidder -d bidder < config/schema.sql
docker-compose exec -T postgres psql -U bidder -d bidder < config/sample_data.sql
make run                                      # Start bidder

# Testing
make test                                     # Run all tests
make test-coverage                            # Generate coverage report
make load-test                                # Run k6 load test

# Get a demo ad response (300x250 mobile banner)
curl -X POST http://localhost:8080/bid -H "Content-Type: application/json" \
  -d '{"id":"test","imp":[{"id":"1","banner":{"w":300,"h":250}}],"site":{"domain":"example.com"},"device":{"geo":{"country":"US"},"devicetype":4}}'

# Docker
make docker-build                             # Build Docker image
make services-up                              # Start all services
make services-down                            # Stop all services

# Kubernetes
kubectl apply -k deploy/k8s/                  # Deploy to Kubernetes
kubectl -n fast-ad-bidder get pods            # Check status

# Monitoring
curl http://localhost:8080/health             # Health check
curl http://localhost:8080/metrics            # Prometheus metrics
open http://localhost:3000                    # Grafana dashboard
```

## Quick Start

### Prerequisites

- **Go**: 1.25.5 or later
- **Docker**: 20.10+ and Docker Compose
- **PostgreSQL**: 14+ (provided via Docker Compose)
- **InfluxDB**: 2.x (optional, provided via Docker Compose)

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd fast-ad-bidder

# Install dependencies
go mod download

# Start infrastructure services (PostgreSQL, InfluxDB, Prometheus, Grafana)
make services-up

# Wait for services to be healthy (about 10 seconds)
sleep 10

# Initialize database schema
docker-compose exec -T postgres psql -U bidder -d bidder < config/schema.sql

# Load sample campaign data
docker-compose exec -T postgres psql -U bidder -d bidder < config/sample_data.sql

# Set up environment variables for local development
cp .env.example .env
# The default .env file is pre-configured for local Docker development
```

### Running the Bidder

```bash
# Development mode (uses .env file for configuration)
make run
# or
go run cmd/bidder/main.go

# Production build
make build
./bin/bidder
```

The bidder will start on port `8080` (configurable via `PORT` environment variable).

### Verify Installation

```bash
# Check health endpoint
curl http://localhost:8080/health
# Expected: {"status":"healthy"}

# Send a test bid request
curl -X POST http://localhost:8080/bid \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-request-001",
    "imp": [{"id": "1", "banner": {"w": 300, "h": 250}}],
    "site": {"id": "site-123", "domain": "example.com"},
    "device": {
      "ua": "Mozilla/5.0",
      "ip": "1.2.3.4",
      "geo": {"country": "US"},
      "devicetype": 4
    }
  }'

# Access monitoring dashboards
# - Grafana: http://localhost:3000 (admin/admin)
# - Prometheus: http://localhost:9090
# - InfluxDB: http://localhost:8086
```

### Getting a Real Ad Response

The sample data includes demo ads. To get a successful bid response:

```bash
# Request a mobile banner ad (300x250) from US
curl -X POST http://localhost:8080/bid \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-mobile-001",
    "imp": [
      {
        "id": "1",
        "banner": {
          "w": 300,
          "h": 250,
          "pos": 1
        }
      }
    ],
    "site": {
      "id": "site-001",
      "domain": "example.com"
    },
    "device": {
      "ua": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)",
      "ip": "1.2.3.4",
      "geo": {
        "country": "US"
      },
      "devicetype": 4
    }
  }' | jq .
```

**Expected Response:**
```json
{
  "id": "test-mobile-001",
  "seatbid": [
    {
      "bid": [
        {
          "id": "bid-...",
          "impid": "1",
          "price": 2.50,
          "adm": "<div style=\"width:300px;height:250px;background:#4285f4;display:flex;align-items:center;justify-content:center;color:white;font-size:24px;\">Sample Ad</div>",
          "adomain": ["advertiser.example.com"],
          "crid": "creative-001",
          "w": 300,
          "h": 250
        }
      ]
    }
  ],
  "cur": "USD"
}
```

**Desktop Banner (728x90):**
```bash
curl -X POST http://localhost:8080/bid \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-desktop-001",
    "imp": [{"id": "1", "banner": {"w": 728, "h": 90}}],
    "site": {"domain": "example.com"},
    "device": {
      "ua": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
      "ip": "1.2.3.4",
      "geo": {"country": "US"},
      "devicetype": 2
    }
  }' | jq .
```

**No-Bid Response** (when no campaigns match):
```json
{
  "id": "test-request-001",
  "nbr": 200,
  "cur": "USD"
}
```

### Endpoints

- **POST /bid** - OpenRTB 2.5 bid request endpoint
- **GET /win** - Win notification callback (query params: `bid`, `price`, `currency`)
- **GET /health** - Health check endpoint
- **GET /metrics** - Prometheus metrics endpoint

## Architecture

### High-Level Overview

```
┌─────────────────┐
│ Google Ad       │
│ Exchange        │
└────────┬────────┘
         │ OpenRTB 2.5 Bid Request
         │ (POST /bid)
         ▼
┌─────────────────────────────────────┐
│  Fast Ad Bidder Service             │
│                                     │
│  ┌──────────────┐  ┌─────────────┐ │
│  │ mTLS Auth    │  │ Validation  │ │
│  │ Middleware   │→ │ Layer       │ │
│  └──────────────┘  └──────┬──────┘ │
│                            │        │
│  ┌─────────────────────────▼──────┐ │
│  │  Campaign Matcher              │ │
│  │  (Geo, Device, Domain, Size)   │ │
│  └─────────────────────────┬──────┘ │
│                            │        │
│  ┌─────────────────────────▼──────┐ │
│  │  Bid Price Calculator          │ │
│  │  (Fixed CPM + Budget Check)    │ │
│  └─────────────────────────┬──────┘ │
│                            │        │
│  ┌─────────────────────────▼──────┐ │
│  │  Response Builder              │ │
│  │  (OpenRTB 2.5 BidResponse)     │ │
│  └────────────────────────────────┘ │
└─────────────────────────────────────┘
         │ OpenRTB 2.5 Bid Response
         ▼
┌─────────────────┐
│ Google Ad       │
│ Exchange        │
└────────┬────────┘
         │ Win Notification
         │ (GET /win?bid=...&price=...)
         ▼
┌─────────────────────────────────────┐
│  Win Processing Pipeline            │
│                                     │
│  ┌──────────────┐  ┌─────────────┐ │
│  │ Bid Cache    │  │ Budget      │ │
│  │ Lookup       │→ │ Deduction   │ │
│  └──────────────┘  └──────┬──────┘ │
│                            │        │
│  ┌─────────────────────────▼──────┐ │
│  │  Metrics Aggregator            │ │
│  │  (Win Rate, CPM, Spend)        │ │
│  └────────────────────────────────┘ │
└─────────────────────────────────────┘
```

### Component Breakdown

#### Core Services

- **Validator** (`src/services/validator/`) - OpenRTB 2.5 request parsing and validation
- **Matcher** (`src/services/matcher/`) - Campaign targeting evaluation (geo, device, domain, creative dimensions)
- **Pricer** (`src/services/pricer/`) - Bid price calculation and budget constraint validation
- **Tracker** (`src/services/tracker/`) - Win notification processing, budget tracking, metrics aggregation
- **Builder** (`src/services/builder/`) - OpenRTB bid response construction

#### Infrastructure

- **Store** (`src/services/store/`) - In-memory campaign cache with PostgreSQL backing (60s reload)
- **Middleware** (`src/middleware/`) - mTLS authentication, structured logging, Prometheus metrics
- **API Handlers** (`src/api/`) - HTTP request handlers for /bid and /win endpoints

#### Data Flow

1. **Bid Request** → mTLS auth → validation → campaign matching → price calculation → response
2. **Win Notification** → bid cache lookup → budget deduction → metrics recording → InfluxDB write

## Configuration

### Environment Variables

Configuration is managed via environment variables. For local development, create a `.env` file:

```bash
# Server Configuration
PORT=8080                          # HTTP server port
LOG_LEVEL=debug                    # Log level (debug, info, warn, error)
METRICS_NAMESPACE=bidder           # Prometheus metrics namespace

# PostgreSQL Configuration (connection string format)
# For local development with Docker Compose:
DATABASE_URL=postgres://bidder:bidder_dev_password@localhost:5432/bidder?sslmode=disable
# For production, use secure credentials and sslmode=require

# InfluxDB Configuration (Optional - for metrics storage)
# Leave empty to disable InfluxDB integration
INFLUXDB_URL=http://localhost:8086
INFLUXDB_TOKEN=dev-token-change-in-production
INFLUXDB_ORG=fast-ad-bidder
INFLUXDB_BUCKET=metrics

# mTLS Configuration
MTLS_ENABLED=false                 # Set to true to enable mutual TLS authentication
TLS_CERT_FILE=config/certs/server.crt
TLS_KEY_FILE=config/certs/server.key
TLS_CA_FILE=config/certs/ca.crt

# Win Notification URL (included in bid responses)
WIN_NOTIFICATION_URL=http://localhost:8080/win
```

**Note**: The bidder uses `github.com/joho/godotenv` to automatically load `.env` file if present. In production (Docker/Kubernetes), use environment variables directly.

### Campaign Configuration

Campaigns are stored in PostgreSQL and loaded into memory on startup with automatic reload every 60 seconds. See `config/schema.sql` for the complete database schema.

**Campaign Table Structure:**
```sql
CREATE TABLE campaigns (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    advertiser_id   TEXT NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('active', 'paused', 'budget_capped')),
    daily_budget    DECIMAL(10,2) NOT NULL CHECK (daily_budget > 0),
    spent_today     DECIMAL(10,2) NOT NULL DEFAULT 0,
    budget_reset_time TIMESTAMPTZ NOT NULL,

    -- Targeting rules stored as JSONB
    targeting       JSONB NOT NULL DEFAULT '{}',

    -- Bid strategy
    bid_strategy    TEXT NOT NULL CHECK (bid_strategy IN ('fixed_cpm', 'dynamic')),
    fixed_cpm       DECIMAL(10,2),
    max_cpm         DECIMAL(10,2),

    -- Creative assignment
    creative_ids    TEXT[] NOT NULL DEFAULT '{}',

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Targeting JSON Structure:**

The targeting JSON uses snake_case keys and specific formats:

```json
{
  "geo": {
    "countries": ["US", "CA", "GB"],
    "regions": ["CA-ON", "NY"]
  },
  "device_types": [2, 4, 5],
  "domains": [],
  "app_bundles": []
}
```

**OpenRTB Device Type Codes:**
- `1` - Mobile/Tablet (general)
- `2` - Personal Computer (desktop)
- `3` - Connected TV
- `4` - Phone
- `5` - Tablet
- `6` - Connected Device
- `7` - Set Top Box

**Example Targeting Scenarios:**

```sql
-- Target US mobile users (phones and tablets)
targeting = '{"geo": {"countries": ["US"], "regions": []}, "device_types": [4, 5], "domains": [], "app_bundles": []}'::jsonb

-- Target desktop users in US, Canada, UK
targeting = '{"geo": {"countries": ["US", "CA", "GB"], "regions": []}, "device_types": [2], "domains": [], "app_bundles": []}'::jsonb

-- Target specific domains (whitelist)
targeting = '{"geo": {"countries": ["US"], "regions": []}, "device_types": [2, 4, 5], "domains": {"whitelist": ["nytimes.com", "wsj.com"]}, "app_bundles": []}'::jsonb
```

**Creative Table Structure:**
```sql
CREATE TABLE creatives (
    id                  TEXT PRIMARY KEY,
    campaign_id         TEXT NOT NULL REFERENCES campaigns(id),
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
```

## Performance & Monitoring

### Latency SLA

- **p95 < 100ms** - 95th percentile bid response latency
- **p99 < 120ms** - 99th percentile bid response latency

Monitor via Prometheus metrics:
```promql
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{endpoint="/bid"}[5m]))
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket{endpoint="/bid"}[5m]))
```

### Key Metrics

**Request Metrics:**
- `http_requests_total{endpoint="/bid",status="200"}` - Successful bid requests
- `http_requests_total{endpoint="/bid",status="400"}` - Validation failures
- `bids_total{type="bid"}` - Bid responses generated
- `bids_total{type="nobid"}` - No-bid responses

**Win Tracking:**
- `wins_received_total` - Win notifications received
- `wins_processed_total` - Win notifications successfully processed
- `wins_orphaned_total` - Orphaned wins (bid not in cache)
- `wins_dropped_total` - Wins dropped due to queue overflow

**Budget & Campaign:**
- `active_campaigns_total` - Active campaigns with available budget
- `campaigns_budget_total` - Total daily budget across all campaigns
- `campaigns_budget_remaining` - Remaining budget across all campaigns
- `campaigns_budget_spent` - Spent budget today
- `campaigns_budget_utilization` - Average budget utilization percentage

### Grafana Dashboards

See `config/grafana/dashboards/bidder.json` for pre-configured Grafana dashboard with:
- Request rate and latency histograms
- Win rate and revenue tracking
- Budget utilization graphs
- Error rate alerts

## Development

### Running Tests

```bash
# Run all tests
make test

# Run with coverage report (generates coverage.html)
make test-coverage

# Run specific test suites
make test-unit           # Unit tests only
make test-integration    # Integration tests only
make test-contract       # Contract tests only

# Run with race detection
make test-all

# Load testing with k6 (requires k6 installed)
make load-test
```

### Code Coverage

**Current Coverage: 58.0%** (core business logic well-covered)

**Coverage by Component:**
- Campaign Matching: ~85%
- Bid Pricing: ~80%
- Budget Tracking: ~75%
- Metrics Aggregation: ~90%
- Request Validation: ~85%
- Win Processing (workers): ~20% (background goroutines)
- I/O operations (DB, InfluxDB): ~10% (tested in integration tests)

**Constitutional Requirement**: 80%+ test coverage for new code

```bash
# Generate detailed coverage report
make test-coverage

# View coverage in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux

# View coverage in terminal
go tool cover -func=coverage.out | grep total
```

**Note**: The coverage tool tracks code in `src/` packages. Integration tests provide additional coverage for I/O operations and background workers not easily tested in unit tests.

### Project Structure

```
.
├── cmd/
│   └── bidder/              # Main application entry point
├── src/
│   ├── api/                 # HTTP handlers
│   ├── lib/                 # Shared utilities
│   ├── middleware/          # HTTP middleware
│   ├── models/              # Data models
│   └── services/            # Business logic
│       ├── builder/         # Response construction
│       ├── matcher/         # Campaign matching
│       ├── pricer/          # Bid pricing
│       ├── store/           # Data storage
│       ├── tracker/         # Win tracking
│       └── validator/       # Request validation
├── tests/
│   ├── contract/            # OpenRTB compliance tests
│   ├── fixtures/            # Test data
│   ├── integration/         # End-to-end tests
│   ├── load/                # k6 load tests
│   └── unit/                # Component tests
├── config/
│   ├── campaigns/           # Campaign definitions
│   ├── certs/              # mTLS certificates (gitignored)
│   ├── grafana/            # Grafana dashboards
│   └── prometheus/         # Prometheus config
├── docker-compose.yml      # Local development services
├── Dockerfile              # Container build
└── README.md               # This file
```

## Deployment

### Docker Compose (Local Development)

The fastest way to run the complete stack locally:

```bash
# Start all services (PostgreSQL, InfluxDB, Prometheus, Grafana, Bidder)
make services-up

# View logs
docker-compose logs -f bidder

# Stop all services
make services-down
```

**Services:**
- Bidder: http://localhost:8080
- Grafana: http://localhost:3000 (admin/admin)
- Prometheus: http://localhost:9090
- InfluxDB: http://localhost:8086

### Docker (Production Build)

```bash
# Build optimized production image (multi-stage, Alpine-based)
make docker-build

# Run container with environment variables
docker run -d \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://user:password@db-host:5432/bidder?sslmode=require" \
  -e INFLUXDB_URL="http://influxdb:8086" \
  -e INFLUXDB_TOKEN="your-token" \
  -e LOG_LEVEL="info" \
  --name fast-ad-bidder \
  fast-ad-bidder:latest

# View logs
docker logs -f fast-ad-bidder

# Check health
docker exec fast-ad-bidder wget -qO- http://localhost:8080/health
```

**Docker Image Details:**
- Base: `golang:1.25-alpine` (builder), `alpine:3.19` (runtime)
- Size: ~20MB compressed
- User: Non-root (bidder:1000)
- Health check: Enabled (30s interval)

### Kubernetes (Production)

Complete deployment manifests are in `deploy/k8s/`:

```bash
# Create namespace
kubectl apply -f deploy/k8s/namespace.yaml

# Deploy configuration (ConfigMap + Secret)
kubectl apply -f deploy/k8s/configmap.yaml
kubectl apply -f deploy/k8s/secret.yaml

# Deploy application
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/service.yaml
kubectl apply -f deploy/k8s/ingress.yaml

# Deploy autoscaling and pod disruption budget
kubectl apply -f deploy/k8s/hpa.yaml
kubectl apply -f deploy/k8s/pdb.yaml

# Optional: Prometheus ServiceMonitor (if using Prometheus Operator)
kubectl apply -f deploy/k8s/servicemonitor.yaml

# Or use Kustomize
kubectl apply -k deploy/k8s/
```

**Kubernetes Configuration:**
- **Replicas**: 3 (min), 20 (max with HPA)
- **HPA**: Auto-scales on 70% CPU or 80% memory utilization
- **PDB**: Minimum 2 pods available during disruptions
- **Resource Requests**: 500m CPU, 512Mi memory
- **Resource Limits**: 2000m CPU, 2Gi memory
- **Probes**: Liveness (20s), Readiness (10s), Startup (10s)
- **Anti-affinity**: Pods spread across different nodes

**Monitoring Setup:**
```bash
# Check deployment status
kubectl -n fast-ad-bidder get pods

# View logs
kubectl -n fast-ad-bidder logs -l app.kubernetes.io/name=fast-ad-bidder -f

# Check metrics
kubectl -n fast-ad-bidder port-forward svc/fast-ad-bidder 8080:80
curl http://localhost:8080/metrics

# Scale manually
kubectl -n fast-ad-bidder scale deployment fast-ad-bidder --replicas=5
```

## Troubleshooting

### Common Issues

**PostgreSQL Connection Failures**

```bash
# Error: "password authentication failed for user bidder"
# Solution: Check DATABASE_URL in .env file

# Verify PostgreSQL is running
docker-compose ps postgres

# Check credentials match docker-compose.yml
grep POSTGRES docker-compose.yml
grep DATABASE_URL .env

# Test connection manually
docker-compose exec postgres psql -U bidder -d bidder -c "SELECT 1"
```

**Database Schema Not Initialized**

```bash
# Error: "column bid_floor_cpm does not exist"
# Solution: Run schema initialization

# Initialize schema
docker-compose exec -T postgres psql -U bidder -d bidder < config/schema.sql

# Verify tables exist
docker-compose exec postgres psql -U bidder -d bidder -c "\dt"

# Load sample data
docker-compose exec -T postgres psql -U bidder -d bidder < config/sample_data.sql
```

**Zero Test Coverage**

```bash
# Error: "coverage: 0.0%"
# Cause: Missing -coverpkg flag for cross-package coverage

# Correct command (use Makefile)
make test-coverage  # Uses -coverpkg=./src/... flag

# Manual command
go test -v -coverprofile=coverage.out -covermode=atomic -coverpkg=./src/... ./...
```

**High Latency (> 100ms p95)**

```bash
# Check active campaign count
curl -s http://localhost:8080/metrics | grep active_campaigns_total

# Reduce to <100 active campaigns for optimal performance
# Review slow queries in logs
docker-compose logs bidder | grep "slow query"

# Check Prometheus histogram
curl -s http://localhost:8080/metrics | grep http_request_duration_seconds
```

**Orphaned Win Notifications**

```bash
# Check orphaned win metric
curl -s http://localhost:8080/metrics | grep wins_orphaned_total

# Common causes:
# 1. Win notification arrives >5 minutes after bid (cache TTL expired)
# 2. Bidder restarted between bid and win (cache cleared)
# 3. Clock drift between bidder instances

# Solution: Increase bid cache TTL or reduce win notification latency
```

**Budget Exhaustion Mid-Day**

```bash
# Check campaign spend
docker-compose exec postgres psql -U bidder -d bidder -c \
  "SELECT id, name, daily_budget, spent_today FROM campaigns WHERE status='active'"

# Verify budget reset scheduler
docker-compose logs bidder | grep "Budget reset scheduler"

# Check reset time (should be midnight UTC)
docker-compose logs bidder | grep "next_reset_at"
```

**mTLS Certificate Errors**

```bash
# Generate self-signed certificates for local testing
cd config/certs
./generate.sh

# Verify certificate validity
openssl x509 -in server.crt -noout -dates
openssl x509 -in server.crt -noout -subject

# Test mTLS connection
curl -v --cert client.crt --key client.key --cacert ca.crt \
  https://localhost:8080/health
```

**Docker Compose Validation Errors**

```bash
# Error: "contains false, which is an invalid type"
# Cause: Boolean values in environment must be strings

# Incorrect:
TLS_ENABLED: false

# Correct:
TLS_ENABLED: "false"

# Validate docker-compose.yml
docker-compose config --quiet
```

**Go Module Download Failures**

```bash
# Error: "go.mod requires go >= 1.25.5"
# Solution: Update Go to 1.25.5 or later

# Check Go version
go version

# Update Go (using Homebrew on macOS/Linux)
brew upgrade go

# Or download from https://go.dev/dl/
```

**Getting No-Bid Responses (nbr: 200)**

```bash
# Problem: Bidder returns no-bid for all requests
# Common causes:

# 1. Incorrect targeting format in database
# Check current targeting format
docker-compose exec postgres psql -U bidder -d bidder -c \
  "SELECT id, targeting FROM campaigns WHERE status='active' LIMIT 1;"

# 2. Fix targeting format (if needed)
docker-compose exec -T postgres psql -U bidder -d bidder << 'EOF'
UPDATE campaigns SET targeting = '{
  "geo": {"countries": ["US"], "regions": []},
  "device_types": [4, 5],
  "domains": [],
  "app_bundles": []
}'::jsonb WHERE id = 'campaign-001';
EOF

# 3. Restart bidder to reload campaigns
pkill -f "go run cmd/bidder/main.go"
make run &

# 4. Wait for campaigns to reload (60 second interval)
sleep 65

# 5. Test with correct device type codes
# Phone (4), Tablet (5), Desktop/PC (2)
curl -X POST http://localhost:8080/bid \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test",
    "imp": [{"id": "1", "banner": {"w": 300, "h": 250}}],
    "site": {"domain": "example.com"},
    "device": {
      "geo": {"country": "US"},
      "devicetype": 4
    }
  }'

# 6. Check metrics for active campaigns
curl -s http://localhost:8080/metrics | grep active_campaigns_total
# Should show: active_campaigns_total 2 (or more)
```

**Targeting Format Requirements**

The bidder expects specific JSON formats for targeting:

```json
// ❌ WRONG - Will not match
{
  "GeoTargeting": ["US"],           // PascalCase keys
  "DeviceTypes": ["mobile"],        // String device types
  "geo": ["US"]                     // Array instead of object
}

// ✅ CORRECT - Will match
{
  "geo": {                          // Object with countries/regions
    "countries": ["US"],
    "regions": []
  },
  "device_types": [4, 5],           // snake_case, numeric codes
  "domains": [],                    // Empty arrays for no targeting
  "app_bundles": []
}
```

## API Examples

### Bid Request Example

```bash
# Successful bid response
curl -X POST http://localhost:8080/bid \
  -H "Content-Type: application/json" \
  -d '{
    "id": "bid-request-123",
    "imp": [
      {
        "id": "1",
        "banner": {
          "w": 300,
          "h": 250,
          "pos": 1
        },
        "bidfloor": 1.0
      }
    ],
    "site": {
      "id": "site-123",
      "domain": "publisher.com",
      "page": "https://publisher.com/article"
    },
    "device": {
      "ua": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)",
      "ip": "1.2.3.4",
      "geo": {
        "country": "US"
      },
      "devicetype": 4
    },
    "user": {
      "id": "user-456"
    }
  }'

# Response (200 OK):
{
  "id": "bid-request-123",
  "seatbid": [
    {
      "bid": [
        {
          "id": "bid-abc-123",
          "impid": "1",
          "price": 2.50,
          "adid": "creative-001",
          "adm": "<div>Ad HTML</div>",
          "adomain": ["advertiser.example.com"],
          "crid": "creative-001",
          "w": 300,
          "h": 250,
          "nurl": "http://localhost:8080/win?bid=bid-abc-123&price=${AUCTION_PRICE}&currency=${AUCTION_CURRENCY}"
        }
      ]
    }
  ],
  "cur": "USD"
}
```

### Win Notification Example

```bash
# Win notification (sent by ad exchange after auction)
curl "http://localhost:8080/win?bid=bid-abc-123&price=2.45&currency=USD"

# Response (200 OK):
{
  "status": "win_recorded",
  "bid_id": "bid-abc-123",
  "price": 2.45,
  "currency": "USD"
}
```

### No-Bid Response

```bash
# When no matching campaigns
curl -X POST http://localhost:8080/bid \
  -H "Content-Type: application/json" \
  -d '{"id": "req-123", "imp": [{"id": "1", "banner": {"w": 300, "h": 250}}]}'

# Response (200 OK):
{
  "id": "req-123",
  "nbr": 200,  # No-bid reason code (200 = no matching creative)
  "cur": "USD"
}
```

### Health Check

```bash
curl http://localhost:8080/health

# Response (200 OK):
{"status":"healthy"}
```

### Prometheus Metrics

```bash
curl http://localhost:8080/metrics

# Sample output:
# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{endpoint="/bid",method="POST",status="200"} 1234
http_requests_total{endpoint="/bid",method="POST",status="204"} 567

# HELP http_request_duration_seconds HTTP request latency
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{endpoint="/bid",le="0.05"} 980
http_request_duration_seconds_bucket{endpoint="/bid",le="0.1"} 1180
http_request_duration_seconds_bucket{endpoint="/bid",le="0.5"} 1234
```

## Additional Resources

- **OpenRTB 2.5 Specification**: https://www.iab.com/guidelines/openrtb/
- **API Documentation**: See `docs/API.md` for complete endpoint documentation
- **Grafana Dashboards**: Pre-configured in `config/grafana/dashboards/`
- **Prometheus Alerts**: Alert rules in `config/prometheus/alerts.yml`
- **Load Test Results**: Run `make load-test` to validate performance

## License

MIT License - See LICENSE file for details

## Support

For issues and questions:
- Open an issue in the repository
- Review troubleshooting section above
- Check logs with `docker-compose logs -f bidder`
