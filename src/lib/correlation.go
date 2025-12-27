package lib

import (
	"github.com/google/uuid"
)

// GenerateCorrelationID generates a unique correlation ID for request tracing
func GenerateCorrelationID() string {
	return uuid.New().String()
}

// CorrelationIDKey is the context key for correlation IDs
const CorrelationIDKey = "correlation_id"
