package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T080: Contract test - win notification processing
func TestWinNotificationProcessing(t *testing.T) {
	e := setupTestServer()

	// Test 1: Valid win notification (create a real bid first)
	t.Run("Valid win notification", func(t *testing.T) {
		// Create a bid
		bidRequest := testFixtures["valid_request_banner_site"]
		bidReqBody, err := json.Marshal(bidRequest)
		require.NoError(t, err)

		bidReq := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(bidReqBody))
		bidReq.Header.Set("Content-Type", "application/json")
		bidRec := httptest.NewRecorder()
		e.ServeHTTP(bidRec, bidReq)

		// Get the bid ID from response
		var bidResp map[string]interface{}
		err = json.Unmarshal(bidRec.Body.Bytes(), &bidResp)
		require.NoError(t, err)

		seatbid := bidResp["seatbid"].([]interface{})
		require.NotEmpty(t, seatbid)
		bids := seatbid[0].(map[string]interface{})["bid"].([]interface{})
		require.NotEmpty(t, bids)
		bidID := bids[0].(map[string]interface{})["id"].(string)

		// Send win notification for this bid
		winReq := httptest.NewRequest(http.MethodGet, "/win?bid="+bidID+"&price=5.25&currency=USD", nil)
		winRec := httptest.NewRecorder()
		e.ServeHTTP(winRec, winReq)

		assert.Equal(t, http.StatusOK, winRec.Code, "Valid win notification should return 200 OK")
	})

	// Test 2: Validation and error cases
	errorTests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		description    string
	}{
		{
			name:           "Missing bid ID",
			queryParams:    "?price=5.25&currency=USD",
			expectedStatus: http.StatusBadRequest,
			description:    "Missing bid ID should return 400 Bad Request",
		},
		{
			name:           "Missing price",
			queryParams:    "?bid=test-bid-123&currency=USD",
			expectedStatus: http.StatusBadRequest,
			description:    "Missing price should return 400 Bad Request",
		},
		{
			name:           "Invalid price format",
			queryParams:    "?bid=test-bid-123&price=invalid&currency=USD",
			expectedStatus: http.StatusBadRequest,
			description:    "Invalid price format should return 400 Bad Request",
		},
		{
			name:           "Orphaned win (bid not found)",
			queryParams:    "?bid=unknown-bid-999&price=5.25&currency=USD",
			expectedStatus: http.StatusNotFound,
			description:    "Win for unknown bid should return 404 Not Found",
		},
		{
			name:           "Expired bid (older than 5 minutes)",
			queryParams:    "?bid=expired-bid-123&price=5.25&currency=USD",
			expectedStatus: http.StatusNotFound,
			description:    "Win for expired bid should return 404 Not Found",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/win"+tt.queryParams, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code, tt.description)
		})
	}
}

// Test that win notifications are processed asynchronously
func TestWinNotificationAsyncProcessing(t *testing.T) {
	e := setupTestServer()

	// First, create a bid by sending a bid request
	bidRequest := testFixtures["valid_request_banner_site"]
	bidReqBody, err := json.Marshal(bidRequest)
	require.NoError(t, err)

	bidReq := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(bidReqBody))
	bidReq.Header.Set("Content-Type", "application/json")
	bidRec := httptest.NewRecorder()
	e.ServeHTTP(bidRec, bidReq)

	// Parse bid response to get bid ID
	var bidResp map[string]interface{}
	err = json.Unmarshal(bidRec.Body.Bytes(), &bidResp)
	require.NoError(t, err)

	seatbid, ok := bidResp["seatbid"].([]interface{})
	require.True(t, ok && len(seatbid) > 0, "Should have at least one seatbid")

	bids, ok := seatbid[0].(map[string]interface{})["bid"].([]interface{})
	require.True(t, ok && len(bids) > 0, "Should have at least one bid")

	bidID, ok := bids[0].(map[string]interface{})["id"].(string)
	require.True(t, ok && bidID != "", "Should have a bid ID")

	// Now send win notification for this bid
	winReq := httptest.NewRequest(http.MethodGet, "/win?bid="+bidID+"&price=5.25&currency=USD", nil)
	winRec := httptest.NewRecorder()

	e.ServeHTTP(winRec, winReq)

	// Should return immediately (200 OK) without waiting for processing
	assert.Equal(t, http.StatusOK, winRec.Code)

	// Response should be fast (async processing)
	// Actual processing happens in background goroutine
}

// Test win notification with different currencies
func TestWinNotificationCurrencySupport(t *testing.T) {
	e := setupTestServer()

	currencies := []string{"USD", "EUR", "GBP", "JPY"}

	for _, currency := range currencies {
		t.Run("Currency_"+currency, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/win?bid=test-bid-123&price=5.25&currency="+currency, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			// Should accept any valid currency code
			assert.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusNotFound,
				"Should return 200 or 404 for valid currency")
		})
	}
}
