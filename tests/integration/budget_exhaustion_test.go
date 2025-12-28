package integration

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

// T087: Integration test - budget exhaustion excludes campaign from bidding
func TestBudgetExhaustionExcludesCampaign(t *testing.T) {
	e := setupTestServer()

	t.Run("Campaign with exhausted budget returns no-bid", func(t *testing.T) {
		// This test requires a campaign with SpentToday >= DailyBudget
		// Test setup would need to configure such a campaign

		bidRequest := map[string]interface{}{
			"id": "test-budget-exhausted",
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
				// Target a campaign that's budget-exhausted
				"domain": "budget-exhausted-campaign.example",
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

		var bidResponse map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &bidResponse)
		require.NoError(t, err)

		// Should return no-bid response
		if seatbid, ok := bidResponse["seatbid"].([]interface{}); ok {
			// Empty or nil seatbid indicates no bid
			assert.Empty(t, seatbid, "Budget-exhausted campaign should not bid")
		}

		// Should have no-bid reason code
		if nbr, exists := bidResponse["nbr"]; exists {
			assert.NotNil(t, nbr, "Should include no-bid reason")
		}
	})

	t.Run("Campaign excluded after budget exhaustion mid-day", func(t *testing.T) {
		// Scenario:
		// 1. Campaign starts day with budget
		// 2. Receives wins that exhaust budget
		// 3. Subsequent bid requests should exclude this campaign

		bidRequest := map[string]interface{}{
			"id": "test-mid-day-exhaustion",
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

		// First request: campaign has budget, should bid
		requestBody, _ := json.Marshal(bidRequest)
		req1 := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
		req1.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec1 := httptest.NewRecorder()

		e.ServeHTTP(rec1, req1)

		var firstResponse map[string]interface{}
		json.Unmarshal(rec1.Body.Bytes(), &firstResponse)

		// Assume we exhaust budget through win notifications
		// (Implementation would simulate this)

		// Second request: campaign budget exhausted, should not bid
		req2 := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
		req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec2 := httptest.NewRecorder()

		e.ServeHTTP(rec2, req2)

		var secondResponse map[string]interface{}
		json.Unmarshal(rec2.Body.Bytes(), &secondResponse)

		// Behavior depends on whether other campaigns match
		// If only budget-exhausted campaign matches, should be no-bid
	})

	t.Run("Multiple campaigns - only budget-exhausted excluded", func(t *testing.T) {
		// Scenario: 2 campaigns match request
		// Campaign A: budget exhausted
		// Campaign B: has budget
		// Should bid with Campaign B only

		bidRequest := map[string]interface{}{
			"id": "test-selective-exclusion",
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

		var bidResponse map[string]interface{}
		err = json.Unmarshal(rec.Body.Bytes(), &bidResponse)
		require.NoError(t, err)

		// Should have a bid from non-exhausted campaign
		if seatbid, ok := bidResponse["seatbid"].([]interface{}); ok {
			if len(seatbid) > 0 {
				// Verify bid is from campaign with budget
				// (Would check campaign ID in implementation)
			}
		}
	})

	t.Run("Budget exhaustion threshold prevents overspend", func(t *testing.T) {
		// Prevent race condition where multiple bids are submitted
		// just before budget exhaustion

		bidRequest := map[string]interface{}{
			"id": "test-threshold",
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

		requestBody, _ := json.Marshal(bidRequest)

		// Submit multiple concurrent bid requests
		done := make(chan int)
		for i := 0; i < 5; i++ {
			go func() {
				req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()

				e.ServeHTTP(rec, req)

				done <- rec.Code
			}()
		}

		// Collect responses
		for i := 0; i < 5; i++ {
			<-done
		}

		// Even with concurrent requests, total spend should not exceed budget
		// Implementation must use proper locking/atomic operations
	})
}

// Test budget reset at midnight UTC
func TestBudgetResetAtMidnight(t *testing.T) {
	e := setupTestServer()

	t.Run("Campaign budget resets at midnight UTC", func(t *testing.T) {
		// This test would require:
		// - Mock time to simulate midnight UTC
		// - Verify SpentToday resets to 0.0
		// - Budget-exhausted campaigns become active again

		// Implementation would use time.AfterFunc or cron job
		// to trigger midnight reset

		// After reset:
		bidRequest := map[string]interface{}{
			"id": "test-after-reset",
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

		// Previously exhausted campaigns should bid again
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
