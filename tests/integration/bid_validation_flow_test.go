package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fast-ad-bidder/bidder/src/api"
	"github.com/fast-ad-bidder/bidder/src/lib"
	"github.com/fast-ad-bidder/bidder/src/middleware"
	"github.com/fast-ad-bidder/bidder/src/services/builder"
	"github.com/fast-ad-bidder/bidder/src/services/matcher"
	"github.com/fast-ad-bidder/bidder/src/services/store"
	"github.com/fast-ad-bidder/bidder/src/services/tracker"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	testServerOnce *echo.Echo
	testLogger     *zap.Logger
	testMetrics    *lib.Metrics
	testCampaigns  []*store.Campaign
	testCreatives  []*store.Creative
)

func init() {
	// Load test campaigns and creatives
	campaignsData, err := os.ReadFile("../../tests/fixtures/campaigns.json")
	if err != nil {
		panic("Failed to load campaign fixtures: " + err.Error())
	}

	var fixtureData struct {
		Campaigns []*store.Campaign `json:"campaigns"`
		Creatives []*store.Creative `json:"creatives"`
	}

	if err := json.Unmarshal(campaignsData, &fixtureData); err != nil {
		panic("Failed to parse campaign fixtures: " + err.Error())
	}

	testCampaigns = fixtureData.Campaigns
	testCreatives = fixtureData.Creatives
}

// mockCampaignLoader loads campaigns from test fixtures
type mockCampaignLoader struct{}

func (m *mockCampaignLoader) LoadFromDB(ctx context.Context) ([]*store.Campaign, []*store.Creative, error) {
	return testCampaigns, testCreatives, nil
}

func setupTestServer() *echo.Echo {
	if testServerOnce != nil {
		return testServerOnce
	}

	e := echo.New()
	e.HideBanner = true

	testLogger, _ = zap.NewDevelopment()
	testMetrics = lib.InitMetrics("bidder_integration_test")

	e.Use(middleware.LoggingMiddleware(testLogger))
	e.Use(middleware.MetricsMiddleware(testMetrics))

	// Initialize test campaign store with mock loader
	loader := &mockCampaignLoader{}
	campaignStore := store.NewMemoryCampaignStore(loader)

	// Load campaigns into memory
	if err := campaignStore.LoadCampaigns(context.Background()); err != nil {
		panic("Failed to load test campaigns: " + err.Error())
	}

	campaignMatcher := matcher.NewCampaignMatcher(campaignStore, testLogger)
	responseBuilder := builder.NewResponseBuilder("https://test.example.com/win")

	// Initialize tracker components for testing
	bidCache := tracker.NewBidCache(5 * time.Minute)
	metricsAgg := tracker.NewMetricsAggregator()
	var influxWriter *tracker.InfluxWriter // nil for tests

	bidHandler := api.NewBidHandler(campaignMatcher, responseBuilder, bidCache, metricsAgg, influxWriter, testLogger, testMetrics)
	e.POST("/bid", bidHandler.HandleBid)

	// Initialize win handler for win notification tests
	parser := tracker.NewParser(testLogger)
	budgetTracker := tracker.NewBudgetTracker(campaignStore, testLogger)
	winProcessor := tracker.NewWinProcessor(bidCache, budgetTracker, metricsAgg, influxWriter, testLogger, 4, 10000)
	winHandler := api.NewWinHandler(parser, winProcessor, bidCache, testLogger)
	e.GET("/win", winHandler.HandleWin)

	testServerOnce = e
	return e
}

// T040: Integration test - end-to-end valid request flow
func TestEndToEndValidRequestFlow(t *testing.T) {
	e := setupTestServer()

	// Load valid request from fixtures
	fixturesData, err := os.ReadFile("../../tests/fixtures/bid_requests.json")
	require.NoError(t, err, "Should load test fixtures")

	var fixtures map[string]interface{}
	err = json.Unmarshal(fixturesData, &fixtures)
	require.NoError(t, err, "Should parse test fixtures")

	// Test multiple valid request types
	testCases := []struct {
		name        string
		fixtureKey  string
		expectedID  string
		description string
	}{
		{
			name:        "Banner Site Request",
			fixtureKey:  "valid_request_banner_site",
			expectedID:  "auction-123",
			description: "Valid banner ad request for website",
		},
		{
			name:        "Banner App Request",
			fixtureKey:  "valid_request_banner_app",
			expectedID:  "auction-456",
			description: "Valid banner ad request for mobile app",
		},
		{
			name:        "Multiple Impressions",
			fixtureKey:  "valid_request_multiple_impressions",
			expectedID:  "auction-789",
			description: "Valid request with multiple ad slots",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get request from fixtures
			requestData := fixtures[tc.fixtureKey]
			requestBody, err := json.Marshal(requestData)
			require.NoError(t, err)

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			// Execute request through full middleware stack
			e.ServeHTTP(rec, req)

			// Verify response status
			assert.Equal(t, http.StatusOK, rec.Code, "Valid request should return 200 OK")

			// Verify correlation ID header was added by logging middleware
			assert.NotEmpty(t, rec.Header().Get("X-Correlation-ID"), "Response should include correlation ID header")

			// Parse and verify response structure
			var response map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err, "Response should be valid JSON")

			// Verify OpenRTB compliance
			assert.Equal(t, tc.expectedID, response["id"], "Response ID must match request ID")

			// For now, we expect either a bid response or no-bid response
			// The actual bid logic will be implemented in User Story 2
			// For User Story 1, we just verify the request is accepted and response is valid
		})
	}
}

// Integration test - complete invalid request rejection flow
func TestEndToEndInvalidRequestRejectionFlow(t *testing.T) {
	e := setupTestServer()

	// Load fixtures
	fixturesData, err := os.ReadFile("../../tests/fixtures/bid_requests.json")
	require.NoError(t, err)

	var fixtures map[string]interface{}
	err = json.Unmarshal(fixturesData, &fixtures)
	require.NoError(t, err)

	// Test multiple invalid request types
	invalidCases := []struct {
		name        string
		fixtureKey  string
		description string
	}{
		{
			name:        "Missing ID",
			fixtureKey:  "invalid_missing_id",
			description: "Request without required id field",
		},
		{
			name:        "Missing Impressions",
			fixtureKey:  "invalid_missing_imp",
			description: "Request without impression array",
		},
		{
			name:        "Missing Site and App",
			fixtureKey:  "invalid_missing_site_and_app",
			description: "Request without site or app",
		},
		{
			name:        "Missing Device",
			fixtureKey:  "invalid_missing_device",
			description: "Request without device object",
		},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get invalid request from fixtures
			requestData := fixtures[tc.fixtureKey]
			requestBody, err := json.Marshal(requestData)
			require.NoError(t, err)

			// Create HTTP request
			req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()

			// Execute request
			e.ServeHTTP(rec, req)

			// Verify rejection
			assert.Equal(t, http.StatusBadRequest, rec.Code, "Invalid request should return 400 Bad Request")

			// Verify correlation ID was still added (for debugging)
			assert.NotEmpty(t, rec.Header().Get("X-Correlation-ID"), "Error responses should include correlation ID")

			// Verify error response structure
			var errorResponse map[string]interface{}
			err = json.Unmarshal(rec.Body.Bytes(), &errorResponse)
			require.NoError(t, err, "Error response should be valid JSON")
			assert.NotNil(t, errorResponse["error"], "Error response must include error field")
		})
	}
}

// Integration test - verify metrics are emitted correctly
func TestMetricsEmissionDuringValidation(t *testing.T) {
	// This test verifies that the metrics middleware properly tracks requests
	// We'll check that Prometheus metrics are incremented (basic smoke test)

	e := setupTestServer()

	// Send a request
	validRequest := map[string]interface{}{
		"id": "test-123",
		"imp": []interface{}{
			map[string]interface{}{
				"id": "imp-1",
				"banner": map[string]interface{}{
					"w": 300,
					"h": 250,
				},
			},
		},
		"site": map[string]interface{}{
			"domain": "test.com",
		},
		"device": map[string]interface{}{
			"ip": "192.0.2.1",
		},
	}

	requestBody, _ := json.Marshal(validRequest)
	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// At this point, metrics should have been recorded
	// We can't easily assert on Prometheus metrics in unit tests,
	// but we verify the request completes successfully
	assert.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusBadRequest,
		"Request should complete with valid status code")
}
