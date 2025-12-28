package contract

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
	testFixtures  map[string]interface{}
	testCampaigns []*store.Campaign
	testCreatives []*store.Creative
)

func init() {
	// Load test fixtures
	data, err := os.ReadFile("../../tests/fixtures/bid_requests.json")
	if err != nil {
		panic("Failed to load test fixtures: " + err.Error())
	}

	if err := json.Unmarshal(data, &testFixtures); err != nil {
		panic("Failed to parse test fixtures: " + err.Error())
	}

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

var testServerOnce *echo.Echo

func setupTestServer() *echo.Echo {
	// Use singleton pattern to avoid re-registering Prometheus metrics
	if testServerOnce != nil {
		return testServerOnce
	}

	e := echo.New()
	e.HideBanner = true

	// Initialize test logger
	logger, _ := zap.NewDevelopment()

	// Initialize test metrics with unique namespace
	metrics := lib.InitMetrics("bidder_contract_test")

	// Add middleware
	e.Use(middleware.LoggingMiddleware(logger))
	e.Use(middleware.MetricsMiddleware(metrics))

	// Initialize test campaign store with mock loader
	loader := &mockCampaignLoader{}
	campaignStore := store.NewMemoryCampaignStore(loader)

	// Load campaigns into memory
	if err := campaignStore.LoadCampaigns(context.Background()); err != nil {
		panic("Failed to load test campaigns: " + err.Error())
	}

	// Log loaded campaigns for debugging
	logger.Info("Test campaigns loaded",
		zap.Int("campaign_count", len(campaignStore.GetAllCampaigns())),
	)

	campaignMatcher := matcher.NewCampaignMatcher(campaignStore, logger)
	responseBuilder := builder.NewResponseBuilder("https://test.example.com/win")

	// Initialize tracker components for testing
	bidCache := tracker.NewBidCache(5 * time.Minute)
	metricsAgg := tracker.NewMetricsAggregator()
	var influxWriter *tracker.InfluxWriter // nil for tests

	// Add bid endpoint
	bidHandler := api.NewBidHandler(campaignMatcher, responseBuilder, bidCache, metricsAgg, influxWriter, logger, metrics)
	e.POST("/bid", bidHandler.HandleBid)

	// Add win endpoint for win notification tests
	parser := tracker.NewParser(logger)
	budgetTracker := tracker.NewBudgetTracker(campaignStore, logger)
	winProcessor := tracker.NewWinProcessor(bidCache, budgetTracker, metricsAgg, influxWriter, logger, 4, 10000)
	winHandler := api.NewWinHandler(parser, winProcessor, bidCache, logger)
	e.GET("/win", winHandler.HandleWin)

	testServerOnce = e
	return e
}

// T034: Contract test - valid bid request accepted
func TestValidBidRequestAccepted(t *testing.T) {
	e := setupTestServer()

	// Get valid request from fixtures
	validRequest := testFixtures["valid_request_banner_site"]
	requestBody, err := json.Marshal(validRequest)
	require.NoError(t, err)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// Execute request
	e.ServeHTTP(rec, req)

	// Assert response
	assert.Equal(t, http.StatusOK, rec.Code, "Valid request should return 200 OK")

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")

	// Verify OpenRTB response structure
	assert.NotNil(t, response["id"], "Response must have id field")
	assert.Equal(t, "auction-123", response["id"], "Response ID should match request ID")
}

// T035: Contract test - missing bid request ID rejected
func TestMissingBidRequestIDRejected(t *testing.T) {
	e := setupTestServer()

	// Get invalid request from fixtures
	invalidRequest := testFixtures["invalid_missing_id"]
	requestBody, err := json.Marshal(invalidRequest)
	require.NoError(t, err)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// Execute request
	e.ServeHTTP(rec, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, rec.Code, "Request missing ID should return 400 Bad Request")

	// Parse error response
	var errorResponse map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &errorResponse)
	require.NoError(t, err, "Error response should be valid JSON")

	// Verify error structure
	assert.NotNil(t, errorResponse["error"], "Response must have error field")
	assert.Contains(t, errorResponse["error"], "id", "Error message should mention missing id field")
}

// T036: Contract test - missing impression array rejected
func TestMissingImpressionArrayRejected(t *testing.T) {
	e := setupTestServer()

	// Get invalid request from fixtures
	invalidRequest := testFixtures["invalid_missing_imp"]
	requestBody, err := json.Marshal(invalidRequest)
	require.NoError(t, err)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// Execute request
	e.ServeHTTP(rec, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, rec.Code, "Request missing impressions should return 400 Bad Request")

	// Parse error response
	var errorResponse map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &errorResponse)
	require.NoError(t, err, "Error response should be valid JSON")

	// Verify error structure
	assert.NotNil(t, errorResponse["error"], "Response must have error field")

	// Check validation errors array
	validationErrors, ok := errorResponse["validation_errors"].([]interface{})
	require.True(t, ok, "Should have validation_errors array")
	require.NotEmpty(t, validationErrors, "Should have at least one validation error")

	// Check that one of the validation errors mentions "imp"
	foundImpError := false
	for _, ve := range validationErrors {
		veMap := ve.(map[string]interface{})
		if field, ok := veMap["field"].(string); ok && (field == "imp" || bytes.Contains([]byte(field), []byte("imp"))) {
			foundImpError = true
			break
		}
	}
	assert.True(t, foundImpError, "Validation errors should mention missing impressions")
}

// T037: Contract test - missing site/app rejected
func TestMissingSiteAndAppRejected(t *testing.T) {
	e := setupTestServer()

	// Get invalid request from fixtures
	invalidRequest := testFixtures["invalid_missing_site_and_app"]
	requestBody, err := json.Marshal(invalidRequest)
	require.NoError(t, err)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// Execute request
	e.ServeHTTP(rec, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, rec.Code, "Request missing site and app should return 400 Bad Request")

	// Parse error response
	var errorResponse map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &errorResponse)
	require.NoError(t, err, "Error response should be valid JSON")

	// Verify error structure
	assert.NotNil(t, errorResponse["error"], "Response must have error field")

	// Check validation errors array
	validationErrors, ok := errorResponse["validation_errors"].([]interface{})
	require.True(t, ok, "Should have validation_errors array")
	require.NotEmpty(t, validationErrors, "Should have at least one validation error")

	// Check that one of the validation errors mentions "site" or "app"
	foundSiteAppError := false
	for _, ve := range validationErrors {
		veMap := ve.(map[string]interface{})
		if field, ok := veMap["field"].(string); ok {
			if bytes.Contains([]byte(field), []byte("site")) || bytes.Contains([]byte(field), []byte("app")) {
				foundSiteAppError = true
				break
			}
		}
	}
	assert.True(t, foundSiteAppError, "Validation errors should mention site or app requirement")
}

// T038: Contract test - missing device object rejected
func TestMissingDeviceObjectRejected(t *testing.T) {
	e := setupTestServer()

	// Get invalid request from fixtures
	invalidRequest := testFixtures["invalid_missing_device"]
	requestBody, err := json.Marshal(invalidRequest)
	require.NoError(t, err)

	// Create HTTP request
	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// Execute request
	e.ServeHTTP(rec, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, rec.Code, "Request missing device should return 400 Bad Request")

	// Parse error response
	var errorResponse map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &errorResponse)
	require.NoError(t, err, "Error response should be valid JSON")

	// Verify error structure
	assert.NotNil(t, errorResponse["error"], "Response must have error field")

	// Check validation errors array
	validationErrors, ok := errorResponse["validation_errors"].([]interface{})
	require.True(t, ok, "Should have validation_errors array")
	require.NotEmpty(t, validationErrors, "Should have at least one validation error")

	// Check that one of the validation errors mentions "device"
	foundDeviceError := false
	for _, ve := range validationErrors {
		veMap := ve.(map[string]interface{})
		if field, ok := veMap["field"].(string); ok && (field == "device" || bytes.Contains([]byte(field), []byte("device"))) {
			foundDeviceError = true
			break
		}
	}
	assert.True(t, foundDeviceError, "Validation errors should mention missing device")
}
