# Tasks: OpenRTB Bidder (Google Services Integration)

**Input**: Design documents from `/specs/001-openrtb-bidder/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), data-model.md, contracts/, research.md, quickstart.md

**Tests**: Constitution mandates Test-First Development (TDD). All test tasks MUST be completed and FAIL before implementation begins.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `src/`, `tests/` at repository root
- Paths assume single project structure from plan.md

---

## Phase 1: Setup (Shared Infrastructure) ✅ COMPLETE

**Purpose**: Project initialization and basic structure

- [x] T001 Initialize Go module with go.mod in repository root
- [x] T002 Create project directory structure (src/, tests/, config/)
- [x] T003 [P] Create src/models directory for entity definitions
- [x] T004 [P] Create src/services directory with subdirectories (validator/, matcher/, pricer/, tracker/)
- [x] T005 [P] Create src/api directory for HTTP handlers
- [x] T006 [P] Create src/middleware directory for mTLS, logging, metrics
- [x] T007 [P] Create src/lib directory for shared utilities
- [x] T008 [P] Create tests/contract directory for OpenRTB compliance tests
- [x] T009 [P] Create tests/integration directory for end-to-end tests
- [x] T010 [P] Create tests/unit directory for component tests
- [x] T011 [P] Create config/campaigns directory for campaign JSON files
- [x] T012 [P] Create config/certs directory for mTLS certificates (gitignored)
- [x] T013 Install Go dependencies: Echo v4, Prebid OpenRTB v19, Zap, Prometheus client
- [x] T014 Install test dependencies: Testify, Pact Go v2
- [x] T015 Create docker-compose.yml with PostgreSQL 14, InfluxDB 2.x, Prometheus, Grafana
- [x] T016 Create PostgreSQL schema in config/schema.sql for campaigns and creatives tables
- [x] T017 Create sample campaign data in config/sample_data.sql for local testing
- [x] T018 Create .env template with environment variables (DB connection, InfluxDB, ports)
- [x] T019 Create .gitignore excluding config/certs/, .env, coverage files

---

## Phase 2: Foundational (Blocking Prerequisites) ✅ COMPLETE

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T020 [P] Create correlation ID generator in src/lib/correlation.go
- [x] T021 [P] Setup Zap structured logger in src/lib/logger.go with JSON formatting
- [x] T022 [P] Create Prometheus metrics registry in src/lib/metrics.go
- [x] T023 [P] Implement mTLS middleware in src/middleware/mtls.go with TLS 1.3 config
- [x] T024 [P] Implement logging middleware in src/middleware/logging.go with correlation ID injection
- [x] T025 [P] Implement metrics middleware in src/middleware/metrics.go for request counting and latency tracking
- [x] T026 Create main application entry point in cmd/bidder/main.go with Echo server setup
- [x] T027 Implement health check endpoint handler in src/api/health.go
- [x] T028 Implement Prometheus metrics endpoint handler in src/api/metrics.go
- [x] T029 Create base campaign store interface in src/services/store/interface.go
- [x] T030 Implement in-memory campaign store with RWMutex in src/services/store/memory.go
- [x] T031 Implement PostgreSQL campaign loader in src/services/store/postgres.go with 60s reload
- [x] T032 Create InfluxDB client wrapper in src/lib/influx.go for async writes

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Receive and Validate Bid Requests (Priority: P1) 🎯 MVP ✅ COMPLETE

**Goal**: Validate incoming OpenRTB 2.5 bid requests from Google Ad Exchange

**Independent Test**: Send sample OpenRTB bid request payloads to /bid endpoint and verify correct validation responses (accept valid requests, reject invalid ones with specific error codes)

### Tests for User Story 1 (Test-First Development - NON-NEGOTIABLE) ⚠️

> **CRITICAL: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T033 [P] [US1] Create OpenRTB 2.5 sample bid requests (valid/invalid) in tests/fixtures/bid_requests.json
- [x] T034 [P] [US1] Contract test: valid bid request accepted in tests/contract/bid_request_validation_test.go
- [x] T035 [P] [US1] Contract test: missing bid request ID rejected in tests/contract/bid_request_validation_test.go
- [x] T036 [P] [US1] Contract test: missing impression array rejected in tests/contract/bid_request_validation_test.go
- [x] T037 [P] [US1] Contract test: missing site/app rejected in tests/contract/bid_request_validation_test.go
- [x] T038 [P] [US1] Contract test: missing device object rejected in tests/contract/bid_request_validation_test.go
- [x] T039 [P] [US1] Contract test: malformed JSON returns 400 error in tests/contract/bid_request_parsing_test.go
- [x] T040 [P] [US1] Integration test: end-to-end valid request flow in tests/integration/bid_validation_flow_test.go

### Implementation for User Story 1

- [x] T041 [P] [US1] Define BidRequest model using Prebid OpenRTB types in src/models/bid_request.go
- [x] T042 [P] [US1] Define BidResponse model using Prebid OpenRTB types in src/models/bid_response.go
- [x] T043 [P] [US1] Define Error response model in src/models/error.go
- [x] T044 [US1] Implement OpenRTB request validator in src/services/validator/validator.go (validate ID, imp, site/app, device)
- [x] T045 [US1] Implement bid request parser in src/services/validator/parser.go with JSON unmarshaling error handling
- [x] T046 [US1] Implement bid endpoint handler in src/api/bid.go (parse request, validate, return 200/400 responses)
- [x] T047 [US1] Add validation error responses with specific error codes in src/models/error.go
- [x] T048 [US1] Wire up /bid POST endpoint in cmd/bidder/main.go with middleware chain
- [x] T049 [US1] Add structured logging for validation failures with correlation IDs in src/services/validator/validator.go
- [x] T050 [US1] Add Prometheus metrics for validation success/failure counters in src/services/validator/validator.go

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently. Bidder accepts valid OpenRTB requests and rejects invalid ones with proper error codes.

---

## Phase 4: User Story 2 - Generate and Submit Bid Responses (Priority: P2)

**Goal**: Evaluate bid requests against campaign inventory and generate OpenRTB bid responses within 100ms p95 latency

**Independent Test**: Process validated bid requests through bid logic, verify bid price calculations match campaign budgets and targeting rules, confirm responses sent within timeout constraints (100-120ms)

### Tests for User Story 2 (Test-First Development - NON-NEGOTIABLE) ⚠️

> **CRITICAL: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T051 [P] [US2] Create sample campaigns and creatives in tests/fixtures/campaigns.json
- [ ] T052 [P] [US2] Contract test: valid bid response schema compliance in tests/contract/bid_response_schema_test.go
- [ ] T053 [P] [US2] Contract test: no-bid response for unmatched request in tests/contract/no_bid_test.go
- [ ] T054 [P] [US2] Unit test: campaign geo targeting matcher in tests/unit/matcher/geo_matcher_test.go
- [ ] T055 [P] [US2] Unit test: campaign device type matcher in tests/unit/matcher/device_matcher_test.go
- [ ] T056 [P] [US2] Unit test: campaign site/app domain matcher in tests/unit/matcher/domain_matcher_test.go
- [ ] T057 [P] [US2] Unit test: creative dimension matcher in tests/unit/matcher/creative_matcher_test.go
- [ ] T058 [P] [US2] Unit test: fixed CPM bid price calculation in tests/unit/pricer/fixed_cpm_test.go
- [ ] T059 [P] [US2] Unit test: budget constraint validation in tests/unit/pricer/budget_check_test.go
- [ ] T060 [P] [US2] Integration test: full bid generation flow with matching campaign in tests/integration/bid_generation_flow_test.go
- [ ] T061 [P] [US2] Integration test: highest bid wins when multiple campaigns match in tests/integration/bid_selection_test.go
- [ ] T062 [US2] Performance test: p95 latency < 100ms validation with k6 in tests/load/bid_latency_test.js

### Implementation for User Story 2

- [ ] T063 [P] [US2] Define Campaign model in src/models/campaign.go with targeting rules and bid strategy
- [ ] T064 [P] [US2] Define Creative model in src/models/creative.go with dimensions and approval status
- [ ] T065 [P] [US2] Define Impression model in src/models/impression.go from OpenRTB types
- [ ] T066 [P] [US2] Implement geo targeting matcher in src/services/matcher/geo.go
- [ ] T067 [P] [US2] Implement device type matcher in src/services/matcher/device.go
- [ ] T068 [P] [US2] Implement site/app domain matcher in src/services/matcher/domain.go
- [ ] T069 [P] [US2] Implement creative dimension matcher in src/services/matcher/creative.go
- [ ] T070 [US2] Implement campaign matcher orchestrator in src/services/matcher/matcher.go (combines all matchers)
- [ ] T071 [US2] Implement fixed CPM pricer in src/services/pricer/fixed_cpm.go
- [ ] T072 [US2] Implement budget validator in src/services/pricer/budget.go (check daily spend < daily budget)
- [ ] T073 [US2] Implement bid response builder in src/services/builder/response.go (construct OpenRTB BidResponse)
- [ ] T074 [US2] Implement no-bid response builder in src/services/builder/nobid.go with reason codes
- [ ] T075 [US2] Update bid endpoint handler in src/api/bid.go to call matcher, pricer, and builder services
- [ ] T076 [US2] Add bid response latency histogram to Prometheus metrics in src/api/bid.go
- [ ] T077 [US2] Add structured logging for bid generation (campaign match, price calculation) in src/services/matcher/matcher.go
- [ ] T078 [US2] Load sample campaigns into PostgreSQL database on startup in cmd/bidder/main.go
- [ ] T079 [US2] Implement campaign reload goroutine with 60s interval in src/services/store/memory.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently. Bidder can receive requests, match campaigns, and return valid bid responses within latency SLA.

---

## Phase 5: User Story 3 - Track Bid Metrics and Win Notifications (Priority: P3)

**Goal**: Process win notifications, track campaign budgets, and emit performance metrics

**Independent Test**: Process bid requests and responses while emitting metrics events, verify win notifications received and matched to original bids, confirm metrics aggregation produces accurate statistics (win rate, spend, impressions served)

### Tests for User Story 3 (Test-First Development - NON-NEGOTIABLE) ⚠️

> **CRITICAL: Write these tests FIRST, ensure they FAIL before implementation**

- [ ] T080 [P] [US3] Contract test: win notification processing in tests/contract/win_notification_test.go
- [ ] T081 [P] [US3] Unit test: bid cache lookup and expiration (5min TTL) in tests/unit/tracker/bid_cache_test.go
- [ ] T082 [P] [US3] Unit test: campaign budget deduction in tests/unit/tracker/budget_deduction_test.go
- [ ] T083 [P] [US3] Unit test: budget-capped status change in tests/unit/tracker/budget_cap_test.go
- [ ] T084 [P] [US3] Unit test: win metrics aggregation (win rate, avg CPM) in tests/unit/tracker/metrics_test.go
- [ ] T085 [P] [US3] Integration test: win notification to budget update flow in tests/integration/win_processing_flow_test.go
- [ ] T086 [P] [US3] Integration test: orphaned win handling (404 response) in tests/integration/orphaned_win_test.go
- [ ] T087 [US3] Integration test: budget exhaustion excludes campaign from bidding in tests/integration/budget_exhaustion_test.go

### Implementation for User Story 3

- [ ] T088 [P] [US3] Define WinNotification model in src/models/win_notification.go
- [ ] T089 [P] [US3] Define BidMetrics model in src/models/bid_metrics.go
- [ ] T090 [P] [US3] Implement bid cache with 5-minute TTL in src/services/tracker/bid_cache.go using sync.Map
- [ ] T091 [US3] Implement win notification parser in src/services/tracker/parser.go (parse query params: bid ID, price, currency)
- [ ] T092 [US3] Implement budget tracker in src/services/tracker/budget.go (update campaign spend in PostgreSQL)
- [ ] T093 [US3] Implement metrics aggregator in src/services/tracker/metrics.go (calculate win rate, avg CPM, total spend)
- [ ] T094 [US3] Implement InfluxDB metrics writer in src/services/tracker/influx.go for async win event writes
- [ ] T095 [US3] Implement win notification endpoint handler in src/api/win.go (lookup bid, queue for processing, return 200/404)
- [ ] T096 [US3] Wire up /win GET endpoint in cmd/bidder/main.go
- [ ] T097 [US3] Add bid ID to cache when bid response generated in src/api/bid.go
- [ ] T098 [US3] Implement async win processor goroutine with buffered channel in src/services/tracker/processor.go
- [ ] T099 [US3] Add Prometheus gauges for active campaigns and budget remaining in src/services/store/memory.go
- [ ] T100 [US3] Add Prometheus counters for wins received in src/api/win.go
- [ ] T101 [US3] Add structured logging for win notifications with correlation from original bid in src/services/tracker/processor.go
- [ ] T102 [US3] Implement budget reset goroutine (midnight UTC) in src/services/tracker/budget.go
- [ ] T103 [US3] Update campaign matcher to exclude budget-capped campaigns in src/services/matcher/matcher.go

**Checkpoint**: All user stories should now be independently functional. Bidder can receive requests, generate bids, and track wins with budget management.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T104 [P] Create README.md in repository root with project overview and setup instructions
- [ ] T105 [P] Create Grafana dashboard JSON in config/grafana/dashboards/bidder.json (request rate, latency, win rate)
- [ ] T106 [P] Create Prometheus alert rules in config/prometheus/alerts.yml (latency > 100ms, error rate > 1%)
- [ ] T107 [P] Create k6 load test script in tests/load/bid_endpoint.js for 1000 QPS validation
- [ ] T108 [P] Add code coverage reporting with go test -cover in tests/
- [ ] T109 [P] Create mTLS certificate generation script in config/certs/generate.sh for local development
- [ ] T110 [P] Document API endpoints in contracts/README.md usage examples
- [ ] T111 [P] Add Docker build configuration in Dockerfile for containerized deployment
- [ ] T112 [P] Create Kubernetes deployment manifests in k8s/ directory (deployment, service, ingress)
- [ ] T113 Run full integration test suite across all user stories in tests/integration/
- [ ] T114 Run k6 load test to validate p95 < 100ms and p99 < 120ms latency at 1000 QPS
- [ ] T115 Validate OpenRTB 2.5 compliance with contract tests in tests/contract/
- [ ] T116 Run quickstart validation: verify all commands in quickstart.md work correctly
- [ ] T117 Review code coverage report and add tests for uncovered paths (target 80%+)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3, 4, 5)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Builds on US1 validation but independently testable with mock validator
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Independently testable with mock bid responses, integrates with US2 for production flow

### Within Each User Story

**Test-First Development Workflow (NON-NEGOTIABLE)**:
1. Write ALL tests for the user story FIRST
2. Run tests and verify they FAIL (red phase)
3. Implement code to make tests pass (green phase)
4. Refactor while keeping tests green
5. Verify 80%+ coverage before marking story complete

**Execution Order**:
- Tests MUST be written and FAIL before implementation
- Models before services (services depend on models)
- Services before API handlers (handlers depend on services)
- Core implementation before integration tasks
- Story complete before moving to next priority

### Parallel Opportunities

**Setup Phase (Phase 1)**:
- T003-T012: All directory creation tasks can run in parallel
- T013-T014: Dependency installation can run in parallel
- T015-T019: Configuration file creation can run in parallel

**Foundational Phase (Phase 2)**:
- T020-T025: All middleware and utility creation can run in parallel
- T029-T032: Storage layer implementation can run in parallel

**User Story 1 Tests**:
- T033-T038: All contract tests can be written in parallel
- T041-T043: All model definitions can be created in parallel

**User Story 2 Tests**:
- T051-T062: All tests can be written in parallel (different files)
- T063-T069: All matchers can be implemented in parallel (different files)

**User Story 3 Tests**:
- T080-T087: All tests can be written in parallel
- T088-T090: Model and cache implementation can run in parallel

**Across User Stories**:
- Once Foundational phase completes, US1, US2, US3 can start in parallel by different team members
- Each story is independently testable and deployable

---

## Parallel Example: User Story 1

```bash
# Launch all test creation tasks together:
Task T033: "Create OpenRTB 2.5 sample bid requests in tests/fixtures/bid_requests.json"
Task T034: "Contract test: valid bid request accepted"
Task T035: "Contract test: missing bid request ID rejected"
# ... all test tasks T033-T040 can run in parallel

# Then launch all model creation tasks together:
Task T041: "Define BidRequest model in src/models/bid_request.go"
Task T042: "Define BidResponse model in src/models/bid_response.go"
Task T043: "Define Error response model in src/models/error.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
   - Write ALL tests (T033-T040)
   - Verify tests FAIL
   - Implement (T041-T050)
   - Verify tests PASS
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready (bidder validates OpenRTB requests)

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP - request validation!)
3. Add User Story 2 → Test independently → Deploy/Demo (full bidding loop!)
4. Add User Story 3 → Test independently → Deploy/Demo (production-ready with metrics!)
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (request validation)
   - Developer B: User Story 2 (bid generation) - can mock validator from US1
   - Developer C: User Story 3 (metrics tracking) - can mock bid responses from US2
3. Stories complete and integrate independently
4. Integration validation in Phase 6

---

## TDD Workflow Checkpoints

Before marking any user story complete, verify:

- [ ] All tests written BEFORE implementation
- [ ] All tests initially FAILED (red phase verified)
- [ ] All tests now PASS (green phase achieved)
- [ ] Code coverage ≥ 80% for user story
- [ ] Independent test criteria met (can demo story standalone)
- [ ] No implementation code committed without corresponding tests

**Constitution Compliance**: Test-First Development is NON-NEGOTIABLE per constitution section II

---

## Notes

- **[P]** tasks = different files, no dependencies, can run in parallel
- **[Story]** label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- **CRITICAL**: Verify tests fail before implementing (TDD red-green-refactor cycle)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Target 80%+ test coverage for new code (constitution requirement)
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence

**Total Tasks**: 117 tasks
**Test Tasks**: 40 tasks (T033-T040, T051-T062, T080-T087)
**Implementation Tasks**: 77 tasks
**Test Coverage**: Constitution mandates 80%+ coverage - all test tasks are MANDATORY
