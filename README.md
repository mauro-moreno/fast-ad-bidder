# Fast Ad Bidder - OpenRTB 2.5 Bidder Service

A high-performance OpenRTB 2.5 bidder service for Google Ad Exchange integration, written in Go. Validates bid requests, matches campaigns with targeting rules, generates bid responses within 100ms p95 latency, and tracks win notifications for budget management.

## Features

- ✅ **OpenRTB 2.5 Compliance** - Full protocol compliance with Google Ad Exchange
- ✅ **Sub-100ms Latency** - p95 < 100ms, p99 < 120ms response times
- ✅ **mTLS Authentication** - Mutual TLS with bidirectional certificate validation
- ✅ **Campaign Targeting** - Geo, device, domain, and creative dimension matching
- ✅ **Budget Management** - Real-time spend tracking with daily budget caps
- ✅ **Win Notification Processing** - Async win tracking with 5-minute bid cache TTL
- ✅ **Production Observability** - Structured logging, Prometheus metrics, InfluxDB integration
- ✅ **1000 QPS Capacity** - Handles 1000 requests/second at peak load

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- PostgreSQL 14+ (via Docker)
- InfluxDB 2.x (optional, via Docker)

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd fast-ad-bidder

# Install dependencies
go mod download

# Start infrastructure services
docker-compose up -d

# Create database schema
psql -h localhost -U bidder -d bidder -f config/schema.sql

# Load sample campaign data
psql -h localhost -U bidder -d bidder -f config/sample_data.sql

# Set up environment variables
cp .env.example .env
# Edit .env with your configuration
```

### Running the Bidder

```bash
# Development mode
go run cmd/bidder/main.go

# Production build
go build -o bin/bidder cmd/bidder/main.go
./bin/bidder
```

The bidder will start on port `8080` (configurable via `PORT` environment variable).

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

```bash
# Server Configuration
PORT=8080                          # HTTP server port
LOG_LEVEL=info                     # Log level (debug, info, warn, error)
METRICS_NAMESPACE=bidder           # Prometheus metrics namespace

# Database Configuration
DATABASE_URL=postgres://bidder:bidder@localhost:5432/bidder?sslmode=disable

# InfluxDB Configuration (Optional)
INFLUXDB_URL=http://localhost:8086
INFLUXDB_TOKEN=<your-token>
INFLUXDB_ORG=bidder
INFLUXDB_BUCKET=metrics

# mTLS Configuration
MTLS_ENABLED=true
TLS_CERT_FILE=config/certs/server.crt
TLS_KEY_FILE=config/certs/server.key
TLS_CA_FILE=config/certs/ca.crt

# Win Notification URL
WIN_NOTIFICATION_URL=https://bidder.example.com/win
```

### Campaign Configuration

Campaigns are stored in PostgreSQL and loaded into memory on startup. See `config/schema.sql` for the database schema.

**Campaign Structure:**
```sql
CREATE TABLE campaigns (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    daily_budget DECIMAL(10,2) NOT NULL,
    spent_today DECIMAL(10,2) DEFAULT 0,
    bid_strategy_type VARCHAR(50) DEFAULT 'FIXED_CPM',
    bid_strategy_value DECIMAL(10,2),
    geo_countries TEXT[],
    geo_regions TEXT[],
    device_types INTEGER[],
    site_domains TEXT[],
    app_bundles TEXT[],
    creative_ids TEXT[]
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
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test suites
go test ./tests/contract/...       # Contract tests
go test ./tests/integration/...    # Integration tests
go test ./tests/unit/...           # Unit tests

# Load testing with k6
k6 run tests/load/bid_endpoint.js
```

### Code Coverage

Target: **80%+ test coverage** for new code (constitutional requirement)

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

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

### Docker

```bash
# Build image
docker build -t fast-ad-bidder:latest .

# Run container
docker run -d \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://..." \
  -e INFLUXDB_URL="http://..." \
  --name bidder \
  fast-ad-bidder:latest
```

### Kubernetes

See `k8s/` directory for deployment manifests:

```bash
# Apply manifests
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/ingress.yaml
```

## Troubleshooting

### Common Issues

**High Latency (> 100ms p95)**
- Check campaign count - reduce to <100 active campaigns
- Verify PostgreSQL connection pool settings
- Review structured logs for slow database queries
- Check Prometheus histogram for latency distribution

**Orphaned Win Notifications**
- Verify bid cache TTL (5 minutes) vs win notification delay
- Check system clock drift between bidder instances
- Review `wins_orphaned_total` metric for patterns

**Budget Exhaustion Mid-Day**
- Check campaign spend patterns in InfluxDB
- Verify daily budget reset at midnight UTC
- Review `campaigns_budget_utilization` gauge

**mTLS Certificate Errors**
- Verify certificate paths in environment variables
- Check certificate expiration: `openssl x509 -in server.crt -noout -dates`
- Ensure CA certificate includes Google Ad Exchange CA

## License

[Your License Here]

## Support

For issues and questions, please open an issue in the repository.
