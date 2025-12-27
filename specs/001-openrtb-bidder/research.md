# Research: OpenRTB Bidder Technology Stack

**Feature**: OpenRTB Bidder (Google Services Integration)
**Date**: 2025-12-27
**Purpose**: Resolve technical unknowns from implementation plan to enable Phase 1 design

## Research Questions

From Technical Context NEEDS CLARIFICATION items:
1. Best language for sub-100ms latency RTB (Go vs Rust vs Python)?
2. HTTP server framework and OpenRTB library?
3. Storage architecture for campaigns and caching?
4. Testing framework for contract and performance validation?

---

## Decision 1: Programming Language

### Decision: Go 1.21+

**Rationale**:
- **Industry Standard**: Prebid Server (most widely deployed header bidding platform) is built in Go and handles production-scale RTB traffic
- **Performance**: Go easily achieves sub-100ms p95 latency. Production Go-based bidders consistently achieve p95 < 50ms with proper optimization
- **OpenRTB Ecosystem**: Best library support with Prebid's actively maintained OpenRTB package (github.com/prebid/openrtb)
- **mTLS Support**: Go's standard library has production-ready TLS/mTLS implementation
- **Concurrency**: Native goroutines provide efficient handling of 1000 QPS without complex async patterns

**Alternatives Considered**:
- **Rust**: Superior raw performance but sparse RTB ecosystem, longer development time, less mature OpenRTB libraries. Considered for hot-path optimization later if needed
- **Python**: Insufficient throughput (3000-3500 req/s per CPU), would require extensive optimization to meet 100ms p95 latency, higher cloud costs at scale

**References**:
- Prebid Server production deployments: https://docs.prebid.org/prebid-server/versions/pbs-versions-go.html
- Google RTB latency requirements: https://developers.google.com/authorized-buyers/rtb/peer-guide

---

## Decision 2: HTTP Framework

### Decision: Echo v4 (github.com/labstack/echo/v4)

**Rationale**:
- **Performance**: Best performance in 2025 benchmarks for real-world scenarios with database interactions and middleware
- **Ecosystem Compatibility**: Built on Go's standard net/http package (unlike Fiber's fasthttp), providing access to vast Go middleware ecosystem
- **Production Features**: Automatic TLS certificate handling, built-in middleware for logging/recovery/monitoring, optimized HTTP router with zero dynamic memory allocation
- **mTLS Support**: Full support for custom TLS configurations via standard Go tls.Config
- **Scalability**: Horizontal scaling design suitable for cloud deployments

**Alternatives Considered**:
- **Fiber**: 10x performance in synthetic benchmarks but limited ecosystem due to fasthttp foundation, middleware compatibility issues
- **Gin**: Similar performance to Echo but Echo showed slight edge in 2025 benchmarks with database interactions

**References**:
- Framework comparison 2025: https://www.buanacoding.com/2025/09/fiber-vs-gin-vs-echo-golang-framework-comparison-2025.html
- Echo documentation: https://echo.labstack.com/

---

## Decision 3: OpenRTB Library

### Decision: Prebid OpenRTB v19+ (github.com/prebid/openrtb/v19)

**Rationale**:
- **Industry Standard**: Maintained by Prebid.org, used by Prebid Server in production deployments handling millions of bid requests
- **Full Specification Support**: Complete OpenRTB 2.5, 3.0, AdCOM 1.0, and Native 1.2 support
- **Active Maintenance**: Prebid took over the original mxmCherry/openrtb project, actively maintained with Go 1.16+ support
- **Schema Validation**: Provides full type definitions for request/response validation
- **Production Proven**: Battle-tested in real-world RTB environments

**Integration Pattern**:
```go
import "github.com/prebid/openrtb/v19/openrtb2"

func handleBidRequest(c echo.Context) error {
    var bidReq openrtb2.BidRequest
    if err := c.Bind(&bidReq); err != nil {
        return c.JSON(400, ErrorResponse{Error: "Invalid OpenRTB request"})
    }
    // Validation and bidding logic
}
```

**References**:
- Prebid OpenRTB: https://github.com/prebid/openrtb

---

## Decision 4: Storage Architecture

### Campaign Data: In-Memory with Periodic Reload

**Decision**: RWMutex-protected Go maps for hot path, PostgreSQL 14+ for cold storage

**Rationale**:
- **Hot Path Requirements**: RTB bidders must preload campaign data into memory to meet 100ms deadlines. PostgreSQL network latency (10ms) can cause 20x throughput drops
- **Read-Heavy Workload**: For 10-100 campaigns with mostly reads, RWMutex-protected maps are optimal (3x faster than sync.Map for this use case)
- **Simple Architecture**: Atomic swap of campaign data every 30-60 seconds via background goroutine
- **ACID Guarantees**: PostgreSQL provides transactional guarantees for campaign budget tracking

**Implementation Pattern**:
```go
type CampaignStore struct {
    mu        sync.RWMutex
    campaigns map[string]*Campaign
    version   int64
}

func (s *CampaignStore) GetCampaign(id string) (*Campaign, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    c, ok := s.campaigns[id]
    return c, ok
}

func (s *CampaignStore) Reload(campaigns []*Campaign) {
    s.mu.Lock()
    defer s.mu.Unlock()
    newMap := make(map[string]*Campaign)
    for _, c := range campaigns {
        newMap[c.ID] = c
    }
    s.campaigns = newMap
    s.version++
}
```

**Alternatives Considered**:
- **Direct PostgreSQL queries**: Too slow for hot path (10ms+ latency kills 100ms budget)
- **sync.Map**: 3x slower than RWMutex for read-heavy workloads, designed for append-only scenarios
- **Redis primary storage**: Unnecessary complexity for 10-100 campaigns, but optional for multi-instance distributed state

### Caching Layer: Redis 7+ (Optional)

**Decision**: Optional Redis for distributed state across multiple bidder instances

**Use Cases**:
- Shared budget counters across bidder instances
- Win rate tracking for real-time budget adjustments
- Rate limiting state

**When Not Needed**: Single-instance deployments at 1000 QPS can skip Redis

### Win Tracking Storage: InfluxDB 2.x

**Decision**: Time-series database for metrics and win notifications

**Rationale**:
- **Industry Standard**: Prebid Server production deployments use InfluxDB for metrics
- **Optimized for Metrics**: Time-series databases optimize for high-volume metric writes
- **Async Processing**: Win notifications processed off critical path via buffered channel
- **Performance**: Sub-ms write latency for time-series data

**Alternatives Considered**:
- **PostgreSQL append-only table**: Simpler stack but less optimized for metric queries
- **Prometheus**: Better for real-time monitoring, but InfluxDB better for historical win data

**References**:
- RTB in-memory requirements: https://github.com/GoogleCloudPlatform/community/blob/master/archived/real-time-bidder/index.md
- PostgreSQL network latency impact: https://www.cybertec-postgresql.com/en/postgresql-network-latency-does-make-a-big-difference/
- sync.Map performance: https://victoriametrics.com/blog/go-sync-map/

---

## Decision 5: Testing Strategy

### Unit Testing: Go stdlib testing + Testify

**Decision**: Go's built-in testing package + github.com/stretchr/testify

**Rationale**:
- **Foundation**: Go's built-in testing package is production-ready
- **Testify Enhancements**: Most popular Go testing framework providing expressive assertions, mocking, test suites, and require package
- **Best Practices**: Table-driven tests for validation logic, interface-based mocking for campaign store

**Example Pattern**:
```go
func TestBidRequestValidation(t *testing.T) {
    tests := []struct {
        name    string
        request openrtb2.BidRequest
        wantErr bool
    }{
        {
            name: "valid request",
            request: openrtb2.BidRequest{ID: "123", Imp: []openrtb2.Imp{{ID: "1"}}},
            wantErr: false,
        },
        {
            name: "missing impression ID",
            request: openrtb2.BidRequest{ID: "123", Imp: []openrtb2.Imp{{}}},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateBidRequest(&tt.request)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Contract Testing: Pact Go v2 + JSON Schema

**Decision**: github.com/pact-foundation/pact-go/v2 + custom OpenRTB schema validation

**Rationale**:
- **Industry Standard**: Pact is standard for HTTP contract testing
- **OpenRTB Compliance**: Validate bid requests/responses against OpenRTB 2.5 specification
- **Consumer-Driven**: Ensures compatibility with Google Ad Exchange expectations

**Hybrid Approach**:
1. Pact for HTTP contract structure
2. Custom JSON Schema validator using OpenRTB 2.5 schema from IAB
3. Integration tests with real OpenRTB payloads from Google examples

### Load Testing: Grafana k6

**Decision**: Grafana k6 for performance validation

**Rationale**:
- **Modern Tool**: Built in Go, designed for performance testing
- **Better UX**: Superior developer experience vs Vegeta with comprehensive documentation
- **Visualization**: Built-in Grafana integration for real-time latency tracking
- **Future-Proof**: Supports HTTP/2, WebSockets, gRPC

**Performance Validation Example**:
```javascript
export let options = {
  stages: [
    { duration: '2m', target: 1000 }, // Ramp to 1000 QPS
  ],
  thresholds: {
    http_req_duration: ['p(95)<100'], // p95 < 100ms requirement
  },
};
```

**References**:
- Testify: https://github.com/stretchr/testify
- Pact documentation: https://docs.pact.io/
- k6: https://k6.io/

---

## Decision 6: Protocol Format

### Decision: JSON for MVP, Protobuf optimization later

**Rationale**:
- **MVP Simplicity**: JSON easier for debugging and development
- **Google Recommendation**: Google recommends Protobuf for RTB due to reduced CPU costs and smaller wire size
- **Migration Path**: Prebid OpenRTB v19 supports both JSON and Protobuf (openrtb2 vs openrtb3 packages)
- **Optimization Timeline**: Add Protobuf support in P3 (metrics tracking phase) or post-MVP if latency targets aren't met with JSON

**Trade-offs**:
- JSON: Easier debugging, larger payload, higher CPU cost
- Protobuf: Better performance, harder debugging, requires .proto file management

---

## Decision 7: mTLS Implementation

### Decision: Go stdlib crypto/tls with TLS 1.3

**Rationale**:
- **Production Ready**: Go's standard library crypto/tls is battle-tested
- **TLS 1.3**: Better performance than TLS 1.2, reduced handshake latency
- **Certificate Management**: Short-lived certificates (30-90 days) with automated rotation

**Implementation Pattern**:
```go
tlsConfig := &tls.Config{
    ClientCAs:  caCertPool,
    ClientAuth: tls.RequireAndVerifyClientCert,
    MinVersion: tls.VersionTLS13,
}
server := &http.Server{
    TLSConfig: tlsConfig,
}
```

**Best Practices**:
- Automate certificate rotation (critical for production)
- Monitor handshake failures via metrics
- Log certificate expiration warnings

**References**:
- mTLS in Go: https://venilnoronha.io/a-step-by-step-guide-to-mtls-in-go
- Certificate management: https://blog.gitguardian.com/mutual-tls-mtls-authentication/

---

## Additional Best Practices

### Monitoring Stack

**Recommendation**:
- **Metrics**: Prometheus with native Go client
- **Logging**: Zap or Zerolog for structured logging
- **Tracing**: OpenTelemetry for correlation IDs
- **Dashboards**: Grafana

**Rationale**: Prebid Server production architecture uses this exact stack, proven at scale

**References**:
- Prebid Server architecture: https://docs.aws.amazon.com/solutions/latest/prebid-server-deployment-on-aws/architecture-details.html

---

## Final Technology Stack Summary

| Component | Technology | Version | Justification |
|-----------|-----------|---------|---------------|
| **Language** | Go | 1.21+ | Production RTB standard, sub-100ms latency, excellent concurrency |
| **HTTP Framework** | Echo | v4 | Best 2025 benchmarks, standard net/http, production features |
| **OpenRTB Library** | Prebid OpenRTB | v19+ | Industry standard, actively maintained, full 2.5 support |
| **Hot Storage** | In-memory (RWMutex) | stdlib | Sub-ms access, optimal for 10-100 campaigns |
| **Cold Storage** | PostgreSQL | 14+ | Campaign management, ACID guarantees for budgets |
| **Cache Layer** | Redis (optional) | 7+ | Distributed state for multi-instance deployments |
| **Metrics DB** | InfluxDB | 2.x | Time-series optimized for win tracking |
| **Unit Testing** | testing + Testify | latest | Standard library + assertions/mocks |
| **Contract Testing** | Pact Go | v2 | OpenRTB compliance validation |
| **Load Testing** | k6 | latest | Latency benchmarking, Grafana integration |
| **Protocol Format** | JSON (MVP) → Protobuf | OpenRTB 2.5 | JSON for dev velocity, Protobuf for production optimization |
| **mTLS** | crypto/tls | stdlib | TLS 1.3, production-ready |
| **Logging** | Zap | latest | Structured logging, high performance |
| **Metrics** | Prometheus | latest | Industry standard, native Go client |

---

## Implementation Priority Alignment

Based on spec user stories:

1. **P1 - Request Validation**: Echo + Prebid OpenRTB + in-memory validation rules
2. **P2 - Bid Generation**: In-memory campaign store + PostgreSQL campaign loader
3. **P3 - Metrics Tracking**: InfluxDB + async win notification processing
4. **Testing**: Testify (unit) → Pact (contract) → k6 (load testing)

---

## Next Steps

With technology decisions finalized, proceed to:
1. **Phase 1 Design**: Create data-model.md defining entity schemas
2. **Phase 1 Contracts**: Generate OpenRTB API contracts in /contracts/
3. **Phase 1 Quickstart**: Document local development setup with Go toolchain
