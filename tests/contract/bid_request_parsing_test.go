package contract

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

// T039: Contract test - malformed JSON returns 400 error
func TestMalformedJSONReturns400(t *testing.T) {
	e := setupTestServer()

	// Create request with malformed JSON
	malformedJSON := []byte("{invalid json }{{{")

	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(malformedJSON))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// Execute request
	e.ServeHTTP(rec, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, rec.Code, "Malformed JSON should return 400 Bad Request")

	// Verify response contains error information
	assert.NotEmpty(t, rec.Body.String(), "Response should contain error message")
	assert.Contains(t, rec.Body.String(), "error", "Response should indicate parsing error")
}

// Additional test: Empty body should return 400
func TestEmptyBodyReturns400(t *testing.T) {
	e := setupTestServer()

	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader([]byte("")))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	// Execute request
	e.ServeHTTP(rec, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, rec.Code, "Empty body should return 400 Bad Request")
}
