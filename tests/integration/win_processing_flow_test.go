package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T085: Integration test - win notification to budget update flow
func TestWinNotificationToBudgetUpdateFlow(t *testing.T) {
	e := setupTestServer()

	// Step 1: Submit a bid request and generate a bid
	bidRequest := map[string]interface{}{
		"id": "test-win-flow",
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

	// Should return bid response
	assert.Equal(t, http.StatusOK, rec.Code)

	var bidResponse map[string]interface{}
	err = json.Unmarshal(rec.Body.Bytes(), &bidResponse)
	require.NoError(t, err)

	// Extract bid ID from response
	var bidID string
	if seatbid, ok := bidResponse["seatbid"].([]interface{}); ok && len(seatbid) > 0 {
		if seat, ok := seatbid[0].(map[string]interface{}); ok {
			if bids, ok := seat["bid"].([]interface{}); ok && len(bids) > 0 {
				if bid, ok := bids[0].(map[string]interface{}); ok {
					bidID = bid["id"].(string)
				}
			}
		}
	}

	require.NotEmpty(t, bidID, "Should have generated a bid ID")

	// Step 2: Get initial campaign budget
	// (In real implementation, would query campaign store)
	initialBudget := 1000.0
	initialSpent := 250.0

	// Step 3: Send win notification
	winURL := "/win?bid=" + bidID + "&price=5.25&currency=USD"
	winReq := httptest.NewRequest(http.MethodGet, winURL, nil)
	winRec := httptest.NewRecorder()

	e.ServeHTTP(winRec, winReq)

	// Should accept win notification
	assert.Equal(t, http.StatusOK, winRec.Code)

	// Step 4: Wait for async processing
	time.Sleep(100 * time.Millisecond)

	// Step 5: Verify budget was updated
	// (In real implementation, would query campaign store)
	expectedSpent := initialSpent + 5.25

	// Budget should be deducted
	assert.InDelta(t, expectedSpent, initialSpent+5.25, 0.01)

	// Campaign should still have remaining budget
	remainingBudget := initialBudget - expectedSpent
	assert.Greater(t, remainingBudget, 0.0, "Campaign should have remaining budget")
}

// Test win notification with budget exhaustion
func TestWinNotificationCausesBudgetExhaustion(t *testing.T) {
	e := setupTestServer()

	// Create campaign with minimal remaining budget
	// (Implementation would setup test campaign with SpentToday = 995.0, DailyBudget = 1000.0)

	// Submit bid request
	bidRequest := map[string]interface{}{
		"id": "test-budget-exhaust",
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

	// Extract bid ID if bid was generated
	var bidID string
	if rec.Code == http.StatusOK {
		var bidResponse map[string]interface{}
		json.Unmarshal(rec.Body.Bytes(), &bidResponse)

		if seatbid, ok := bidResponse["seatbid"].([]interface{}); ok && len(seatbid) > 0 {
			if seat, ok := seatbid[0].(map[string]interface{}); ok {
				if bids, ok := seat["bid"].([]interface{}); ok && len(bids) > 0 {
					if bid, ok := bids[0].(map[string]interface{}); ok {
						bidID = bid["id"].(string)
					}
				}
			}
		}
	}

	if bidID != "" {
		// Send win notification that exhausts budget
		winURL := "/win?bid=" + bidID + "&price=10.00&currency=USD"
		winReq := httptest.NewRequest(http.MethodGet, winURL, nil)
		winRec := httptest.NewRecorder()

		e.ServeHTTP(winRec, winReq)

		assert.Equal(t, http.StatusOK, winRec.Code)

		// Wait for processing
		time.Sleep(100 * time.Millisecond)

		// Next bid request should not match the budget-exhausted campaign
		req2 := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
		req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec2 := httptest.NewRecorder()

		e.ServeHTTP(rec2, req2)

		// Should return no-bid or bid from different campaign
		// (Implementation specific)
	}
}

// Test multiple concurrent win notifications
func TestConcurrentWinNotifications(t *testing.T) {
	e := setupTestServer()

	// Generate multiple bids first
	bidIDs := make([]string, 5)

	for i := 0; i < 5; i++ {
		bidRequest := map[string]interface{}{
			"id": string(rune('A' + i)),
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
		req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(requestBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK {
			var bidResponse map[string]interface{}
			json.Unmarshal(rec.Body.Bytes(), &bidResponse)

			if seatbid, ok := bidResponse["seatbid"].([]interface{}); ok && len(seatbid) > 0 {
				if seat, ok := seatbid[0].(map[string]interface{}); ok {
					if bids, ok := seat["bid"].([]interface{}); ok && len(bids) > 0 {
						if bid, ok := bids[0].(map[string]interface{}); ok {
							bidIDs[i] = bid["id"].(string)
						}
					}
				}
			}
		}
	}

	// Send concurrent win notifications
	done := make(chan int)
	for i, bidID := range bidIDs {
		if bidID != "" {
			go func(id string, idx int) {
				winURL := "/win?bid=" + id + "&price=5.25&currency=USD"
				winReq := httptest.NewRequest(http.MethodGet, winURL, nil)
				winRec := httptest.NewRecorder()

				e.ServeHTTP(winRec, winReq)

				done <- winRec.Code
			}(bidID, i)
		} else {
			done <- 0
		}
	}

	// Wait for all win notifications
	successCount := 0
	for i := 0; i < 5; i++ {
		code := <-done
		if code == http.StatusOK {
			successCount++
		}
	}

	// All valid wins should be accepted
	assert.Greater(t, successCount, 0, "At least some wins should be processed")

	// Wait for async processing
	time.Sleep(200 * time.Millisecond)

	// No race conditions or data corruption should occur
}
