package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T052: Contract test - valid bid response schema compliance
func TestValidBidResponseSchemaCompliance(t *testing.T) {
	e := setupTestServer()

	// Create a valid bid request
	bidRequest := map[string]interface{}{
		"id": "test-auction-123",
		"imp": []interface{}{
			map[string]interface{}{
				"id": "imp-1",
				"banner": map[string]interface{}{
					"w": 300,
					"h": 250,
				},
				"bidfloor": 1.00,
			},
		},
		"site": map[string]interface{}{
			"domain": "example.com",
		},
		"device": map[string]interface{}{
			"ip":         "192.0.2.1",
			"devicetype": 2,
			"geo": map[string]interface{}{
				"country": "USA",
			},
		},
	}

	requestBody, err := json.Marshal(bidRequest)
	require.NoError(t, err)

	// Send request
	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, rec.Code, "Valid request should return 200 OK")

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")

	// Debug: print response
	t.Logf("Response JSON: %s", rec.Body.String())

	// Validate OpenRTB 2.5 BidResponse schema
	assert.Equal(t, "test-auction-123", response["id"], "Response ID must match request ID")
	assert.NotNil(t, response["seatbid"], "Response must have seatbid field")
	assert.NotNil(t, response["cur"], "Response must have currency field")

	// Check seatbid structure
	seatbid, ok := response["seatbid"].([]interface{})
	require.True(t, ok, "seatbid must be array")

	// If there's a bid, validate its structure
	if len(seatbid) > 0 {
		seat := seatbid[0].(map[string]interface{})
		bids, ok := seat["bid"].([]interface{})
		require.True(t, ok, "seat must have bid array")

		if len(bids) > 0 {
			bid := bids[0].(map[string]interface{})

			// Validate required bid fields
			assert.NotNil(t, bid["id"], "Bid must have id")
			assert.NotNil(t, bid["impid"], "Bid must have impid")
			assert.NotNil(t, bid["price"], "Bid must have price")
			assert.NotNil(t, bid["adid"], "Bid must have adid")
			assert.NotNil(t, bid["crid"], "Bid must have crid (creative ID)")
			assert.NotNil(t, bid["w"], "Bid must have width")
			assert.NotNil(t, bid["h"], "Bid must have height")

			// Validate price is positive
			price, ok := bid["price"].(float64)
			if ok {
				assert.Greater(t, price, 0.0, "Bid price must be positive")
			}

			// Validate dimensions match impression
			assert.Equal(t, float64(300), bid["w"], "Bid width should match impression")
			assert.Equal(t, float64(250), bid["h"], "Bid height should match impression")
		}
	}
}

// T053: Contract test - no-bid response for unmatched request
func TestNoBidResponseForUnmatchedRequest(t *testing.T) {
	e := setupTestServer()

	// Create a bid request that won't match any campaigns
	// (using a domain/geo/device that no campaign targets)
	bidRequest := map[string]interface{}{
		"id": "test-auction-nobid",
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
			"domain": "no-match-domain.example",
		},
		"device": map[string]interface{}{
			"ip":         "192.0.2.1",
			"devicetype": 2,
			"geo": map[string]interface{}{
				"country": "ZZZ", // Non-existent country
			},
		},
	}

	requestBody, err := json.Marshal(bidRequest)
	require.NoError(t, err)

	// Send request
	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return 200 OK (no-bid is still a valid response)
	assert.Equal(t, http.StatusOK, rec.Code, "No-bid should return 200 OK")

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err, "Response should be valid JSON")

	// Debug: print response
	t.Logf("No-bid Response JSON: %s", rec.Body.String())

	// Validate no-bid response
	assert.Equal(t, "test-auction-nobid", response["id"], "Response ID must match request ID")

	// Check seatbid - can be omitted or empty array per OpenRTB spec
	if seatbidRaw, exists := response["seatbid"]; exists {
		seatbid, ok := seatbidRaw.([]interface{})
		require.True(t, ok, "seatbid must be array if present")
		assert.Empty(t, seatbid, "No-bid response should have empty seatbid array")
	}

	// Check for no-bid reason code
	if nbr, exists := response["nbr"]; exists {
		nbrFloat, ok := nbr.(float64)
		if ok {
			assert.Greater(t, nbrFloat, float64(0), "No-bid reason code should be set")
		}
	}
}
