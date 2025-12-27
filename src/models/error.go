package models

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error            string            `json:"error"`
	ErrorCode        string            `json:"error_code,omitempty"`
	ValidationErrors []ValidationError `json:"validation_errors,omitempty"`
	CorrelationID    string            `json:"correlation_id,omitempty"`
}

// ValidationError represents a field-level validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error codes for validation failures
const (
	ErrorCodeMissingID         = "MISSING_REQUEST_ID"
	ErrorCodeMissingImps       = "MISSING_IMPRESSIONS"
	ErrorCodeEmptyImps         = "EMPTY_IMPRESSIONS"
	ErrorCodeMissingSiteAndApp = "MISSING_SITE_AND_APP"
	ErrorCodeBothSiteAndApp    = "BOTH_SITE_AND_APP"
	ErrorCodeMissingDevice     = "MISSING_DEVICE"
	ErrorCodeInvalidJSON       = "INVALID_JSON"
	ErrorCodeInternalError     = "INTERNAL_ERROR"
)

// NewErrorResponse creates a new error response
func NewErrorResponse(message string, code string) *ErrorResponse {
	return &ErrorResponse{
		Error:     message,
		ErrorCode: code,
	}
}

// NewValidationErrorResponse creates a new validation error response
func NewValidationErrorResponse(message string, code string, validationErrors []ValidationError) *ErrorResponse {
	return &ErrorResponse{
		Error:            message,
		ErrorCode:        code,
		ValidationErrors: validationErrors,
	}
}

// WithCorrelationID adds correlation ID to error response
func (e *ErrorResponse) WithCorrelationID(correlationID string) *ErrorResponse {
	e.CorrelationID = correlationID
	return e
}
