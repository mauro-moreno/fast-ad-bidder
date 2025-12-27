package validator

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/fast-ad-bidder/bidder/src/models"
	"go.uber.org/zap"
)

// Parser handles parsing of OpenRTB bid requests
type Parser struct {
	logger *zap.Logger
}

// NewParser creates a new bid request parser
func NewParser(logger *zap.Logger) *Parser {
	return &Parser{
		logger: logger,
	}
}

// ParseRequest parses JSON bid request from request body
func (p *Parser) ParseRequest(body io.Reader, correlationID string) (*models.BidRequest, *models.ErrorResponse) {
	var bidRequest models.BidRequest

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields() // Strict parsing

	if err := decoder.Decode(&bidRequest); err != nil {
		// Log parsing error
		p.logger.Warn("Failed to parse bid request JSON",
			zap.String("correlation_id", correlationID),
			zap.Error(err),
		)

		// Determine error type
		errorMsg := "Invalid JSON format"
		if err == io.EOF {
			errorMsg = "Request body is empty"
		} else if syntaxErr, ok := err.(*json.SyntaxError); ok {
			errorMsg = fmt.Sprintf("JSON syntax error at position %d", syntaxErr.Offset)
		} else if unmarshalErr, ok := err.(*json.UnmarshalTypeError); ok {
			errorMsg = fmt.Sprintf("Invalid type for field '%s': expected %s", unmarshalErr.Field, unmarshalErr.Type)
		}

		return nil, models.NewErrorResponse(errorMsg, models.ErrorCodeInvalidJSON).WithCorrelationID(correlationID)
	}

	return &bidRequest, nil
}

// ParseJSON is a helper function for parsing JSON from byte array
func (p *Parser) ParseJSON(data []byte, correlationID string) (*models.BidRequest, *models.ErrorResponse) {
	var bidRequest models.BidRequest

	if err := json.Unmarshal(data, &bidRequest); err != nil {
		p.logger.Warn("Failed to parse bid request JSON",
			zap.String("correlation_id", correlationID),
			zap.Error(err),
		)

		return nil, models.NewErrorResponse("Invalid JSON format", models.ErrorCodeInvalidJSON).WithCorrelationID(correlationID)
	}

	return &bidRequest, nil
}
