# Implementation Plan: OpenRTB Bidder (Google Services Integration)

**Branch**: `001-openrtb-bidder` | **Date**: 2025-12-27 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-openrtb-bidder/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Build an OpenRTB 2.5 bidder service to receive bid requests from Google Ad Exchange, validate requests, match against campaign inventory, generate bid responses within 100ms p95 latency, and track win notifications for budget management. Initial scope: 1000 QPS, mTLS authentication, banner ads only. Three prioritized user stories: (P1) request validation, (P2) bid generation, (P3) metrics tracking.

## Technical Context

**Language/Version**: Go 1.21+ (production RTB standard, achieves sub-50ms p95 latency, excellent concurrency via goroutines)
**Primary Dependencies**: Echo v4 (HTTP framework), Prebid OpenRTB v19+ (OpenRTB 2.5 library), Zap (structured logging), Prometheus (metrics)
**Storage**: In-memory RWMutex-protected maps (hot path campaign data), PostgreSQL 14+ (cold storage for campaign management), InfluxDB 2.x (win tracking/metrics), Redis 7+ optional (multi-instance distributed state)
**Testing**: Go testing + Testify (unit tests), Pact Go v2 (contract testing for OpenRTB compliance), k6 (load testing for latency validation)
**Target Platform**: Linux server (containerized deployment for cloud environments)
**Project Type**: single (bidder service - single deployable unit per constitution)
**Performance Goals**: 1000 requests/second at peak, p95 latency < 100ms, p99 < 120ms for bid processing
**Constraints**: <100ms p95 response time (hard requirement from Google Ad Exchange), mTLS certificate management, OpenRTB 2.5 schema compliance
**Scale/Scope**: 1000 QPS initially, ~10-100 active campaigns, banner creatives only, single-region deployment

**Research**: See [research.md](research.md) for detailed technology selection rationale

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Verify compliance with `.specify/memory/constitution.md`:

### I. Service-Oriented Architecture ✓
- [x] Service has well-defined API contract (OpenAPI/gRPC schema) - OpenRTB 2.5 spec is the contract, will document in contracts/
- [x] Service can be deployed independently - Bidder is standalone service, no cross-service dependencies
- [x] Service boundaries align with business capability (not technical layer) - "Bid Request Processing" is a clear business capability
- [x] Cross-service dependencies are explicit and versioned - External dependency only: Google Ad Exchange (versioned via OpenRTB 2.5)
- [x] Communication protocols documented (REST/gRPC/messaging) - HTTP POST (OpenRTB over HTTP), will document request/response formats

### II. Test-First Development ✓
- [x] Test plan includes contract tests for all APIs - OpenRTB request/response validation tests required (FR-002, FR-003, FR-006)
- [x] Test plan includes integration tests for service interactions - Will test full request→validation→bidding→response flow
- [x] TDD workflow enforced (red → green → refactor) - Spec defines acceptance scenarios before implementation
- [x] Target: 80%+ test coverage for new code - Critical path (validation + bidding logic) requires comprehensive coverage

### III. Performance & Observability ✓
- [x] Health check endpoint planned - Required for deployment per constitution quality gates
- [x] Structured logging with correlation IDs planned - FR-012 mandates correlation IDs for request tracing
- [x] Metrics endpoints planned (latency, throughput, errors) - User Story 3 (P3) covers metrics collection
- [x] Performance SLA defined (target: p95 < 100ms for bid processing) - SC-001: p95 < 100ms, p99 < 120ms
- [x] Alert thresholds defined for critical paths - Will define for: timeout rate, win rate, budget exhaustion, error rate

**Violations**: None - all constitutional requirements satisfied.

**Constitution Compliance Status**: ✅ PASSED - Ready for Phase 0 research

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
src/
├── models/               # Data models for entities (BidRequest, BidResponse, Campaign, etc.)
├── services/            # Business logic (validation, bid calculation, win tracking)
│   ├── validator/       # OpenRTB request validation
│   ├── matcher/         # Campaign matching and targeting
│   ├── pricer/          # Bid price calculation
│   └── tracker/         # Win notification handling, metrics
├── api/                 # HTTP handlers for bid endpoint, win notifications
├── middleware/          # mTLS authentication, logging, metrics collection
└── lib/                 # Shared utilities (OpenRTB parsing, correlation ID generation)

tests/
├── contract/            # OpenRTB 2.5 schema compliance tests
├── integration/         # End-to-end bidding flow tests
└── unit/                # Component-level tests

config/                  # Configuration management
├── campaigns/           # Campaign definitions (JSON/YAML)
└── certs/              # mTLS certificates (not committed, environment-specific)
```

**Structure Decision**: Single project structure selected. This is a standalone bidder service with no frontend or mobile components. All bidding logic is contained in one deployable unit following the service-oriented architecture principle. The structure separates concerns by layer (models, services, api) while maintaining a simple, monolithic deployment suitable for 1000 QPS scale.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |


---

## Phase 1 Completion: Post-Design Constitution Re-Check

*Required: Re-evaluate after design artifacts generated*

### I. Service-Oriented Architecture ✓
- [x] **API Contract Delivered**: OpenAPI 3.0 spec in `contracts/openapi.yaml` documents all endpoints (/bid, /win, /health, /metrics)
- [x] **Independent Deployment**: Single deployable service with no runtime dependencies on other services
- [x] **Business Capability Alignment**: "Real-Time Bid Processing" is clear business function
- [x] **Documented Protocols**: HTTP POST with OpenRTB 2.5 JSON payload, mTLS authentication
- [x] **Versioned Dependencies**: External dependency only Google Ad Exchange (OpenRTB 2.5 protocol version)

### II. Test-First Development ✓
- [x] **Contract Tests Planned**: Pact Go framework specified in research.md, OpenRTB schema validation defined
- [x] **Integration Tests Planned**: End-to-end bid flow tests outlined in quickstart.md
- [x] **TDD Workflow**: Unit test examples provided in research.md using table-driven tests
- [x] **80% Coverage Target**: Testify assertions framework selected for comprehensive test coverage

### III. Performance & Observability ✓
- [x] **Health Check Implemented**: `/health` endpoint defined in openapi.yaml with campaign load status
- [x] **Structured Logging**: Zap framework selected in research.md for JSON logging with correlation IDs
- [x] **Metrics Exposed**: `/metrics` endpoint defined with Prometheus histogram for latency tracking
- [x] **SLA Defined**: p95 < 100ms, p99 < 120ms documented in Technical Context and validated via k6 load tests
- [x] **Alerts Planned**: Threshold monitoring defined in quickstart.md (latency, error rate, budget exhaustion)

**Post-Design Violations**: None - all constitutional requirements satisfied with concrete implementation plans

**Constitution Compliance Status**: ✅ PASSED - Ready for Phase 2 (task generation via /speckit.tasks)

---

## Planning Summary

### Artifacts Generated

**Phase 0: Research**
- `research.md` - Technology stack selection with industry research and rationale

**Phase 1: Design**
- `data-model.md` - Complete entity schemas with validation rules and storage strategy
- `contracts/openapi.yaml` - OpenAPI 3.0 specification for all HTTP endpoints
- `contracts/README.md` - API contract documentation with examples and testing guidance
- `quickstart.md` - Local development setup, testing guide, and troubleshooting

**Agent Context**
- `CLAUDE.md` - Updated with Go, Echo, PostgreSQL, InfluxDB technology stack

### Technology Stack Finalized

| Layer | Technology | Rationale |
|-------|-----------|-----------|
| Language | Go 1.21+ | Production RTB standard, sub-50ms latency |
| HTTP Framework | Echo v4 | Best 2025 benchmarks, standard net/http compatibility |
| OpenRTB Library | Prebid OpenRTB v19 | Industry standard, full 2.5 support |
| Hot Storage | In-memory RWMutex maps | Sub-ms access for campaign data |
| Cold Storage | PostgreSQL 14+ | ACID guarantees for budgets |
| Metrics | InfluxDB 2.x + Prometheus | Time-series wins + real-time metrics |
| Testing | Testify + Pact + k6 | Unit + contract + load testing |
| Auth | crypto/tls (mTLS) | TLS 1.3, production-ready |

### Next Steps

1. **Run `/speckit.tasks`** to generate dependency-ordered implementation tasks from this plan
2. **Start with P1 User Story**: Request validation (highest priority MVP)
3. **Follow TDD workflow**: Write failing tests → Implement → Refactor
4. **Track progress**: Use todo list to mark tasks complete

### Design Decisions

**Key Architectural Choices**:
- In-memory campaign storage with 60-second reload (meets 100ms latency requirement)
- Async win notification processing (off critical path)
- JSON protocol for MVP, Protobuf optimization later
- Single-instance deployment at 1000 QPS (horizontal scaling via Kubernetes if needed)

**Performance Strategy**:
- Campaign data preloaded in memory (no database I/O on hot path)
- RWMutex-protected maps (3x faster than sync.Map for read-heavy workload)
- Prometheus histogram buckets aligned with SLA thresholds (10ms, 50ms, 100ms, 150ms, 200ms)

**Testing Strategy**:
- Contract tests validate OpenRTB 2.5 compliance
- Integration tests cover full bid flow
- Load tests validate 1000 QPS at p95 < 100ms
- k6 results feed into InfluxDB for latency trend analysis

---

**Planning Status**: ✅ COMPLETE - All design artifacts generated and constitution compliance verified
