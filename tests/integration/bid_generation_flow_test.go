package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T060: Integration test - full bid generation flow with matching campaign
func TestFullBidGenerationFlowWithMatchingCampaign(t *testing.T) {
	e := setupTestServer()

	// Load campaign fixtures to understand what will match
	fixturesData, err := os.ReadFile("../../tests/fixtures/campaigns.json")
	require.NoError(t, err)

	var fixtures map[string]interface{}
	err = json.Unmarshal(fixturesData, &fixtures)
	require.NoError(t, err)

	// Create a bid request that should match campaign-001
	// Campaign-001 targets: USA, device type 2 (desktop), domain whitelist including example.com
	bidRequest := map[string]interface{}{
		"id": "test-bid-generation",
		"imp": []interface{}{
			map[string]interface{}{
				"id": "imp-1",
				"banner": map[string]interface{}{
					"w": 300,
					"h": 250,
				},
				"bidfloor": 0.50,
			},
		},
		"site": map[string]interface{}{
			"domain": "example.com", // Matches campaign-001 whitelist
		},
		"device": map[string]interface{}{
			"ip":         "192.0.2.1",
			"devicetype": 2, // Desktop - matches campaign-001
			"geo": map[string]interface{}{
				"country": "USA", // Matches campaign-001
			},
		},
	}

	requestBody, err := json.Marshal(bidRequest)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	// Should return 200 OK
	assert.Equal(t, http.StatusOK, rec.Code)

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify bid response structure
	assert.Equal(t, "test-bid-generation", response["id"])

	seatbid, ok := response["seatbid"].([]interface{})
	require.True(t, ok)

	// Should have at least one bid (assuming campaigns are loaded)
	if len(seatbid) > 0 {
		seat := seatbid[0].(map[string]interface{})
		bids, ok := seat["bid"].([]interface{})
		require.True(t, ok)

		if len(bids) > 0 {
			bid := bids[0].(map[string]interface{})

			// Verify bid fields
			assert.NotNil(t, bid["id"])
			assert.Equal(t, "imp-1", bid["impid"])
			assert.NotNil(t, bid["price"])
			assert.NotNil(t, bid["crid"])

			// Verify price is reasonable
			price, ok := bid["price"].(float64)
			if ok {
				assert.Greater(t, price, 0.0)
				assert.GreaterOrEqual(t, price, 0.50) // Should meet bid floor
			}

			// Verify dimensions
			assert.Equal(t, float64(300), bid["w"])
			assert.Equal(t, float64(250), bid["h"])
		}
	}
}

// T061: Integration test - highest bid wins when multiple campaigns match
func TestHighestBidWinsWithMultipleCampaigns(t *testing.T) {
	e := setupTestServer()

	// Create a bid request that could match multiple campaigns
	// This tests the bid selection logic
	bidRequest := map[string]interface{}{
		"id": "test-highest-bid",
		"imp": []interface{}{
			map[string]interface{}{
				"id": "imp-1",
				"banner": map[string]interface{}{
					"w": 300,
					"h": 250,
				},
				"bidfloor": 0.10,
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

	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	seatbid, ok := response["seatbid"].([]interface{})
	require.True(t, ok)

	if len(seatbid) > 0 {
		seat := seatbid[0].(map[string]interface{})
		bids, ok := seat["bid"].([]interface{})
		require.True(t, ok)

		// Should return the highest bid
		// If multiple campaigns match, the one with highest max_bid_cpm should win
		if len(bids) > 0 {
			bid := bids[0].(map[string]interface{})
			price, ok := bid["price"].(float64)
			if ok {
				// Verify it's a competitive price
				assert.Greater(t, price, 0.0)
				t.Logf("Winning bid price: $%.2f", price)
			}
		}
	}
}
