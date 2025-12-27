package validator

import (
	"fmt"

	"github.com/fast-ad-bidder/bidder/src/lib"
	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// Validator validates OpenRTB 2.5 bid requests
type Validator struct {
	logger  *zap.Logger
	metrics *lib.Metrics
}

// NewValidator creates a new bid request validator
func NewValidator(logger *zap.Logger, metrics *lib.Metrics) *Validator {
	return &Validator{
		logger:  logger,
		metrics: metrics,
	}
}

// ValidateRequest validates an OpenRTB bid request against required fields
func (v *Validator) ValidateRequest(req *models.BidRequest, correlationID string) *models.ErrorResponse {
	var validationErrors []models.ValidationError

	// Validate request ID (required)
	if req.ID == "" {
		validationErrors = append(validationErrors, models.ValidationError{
			Field:   "id",
			Message: "Request ID is required",
		})
	}

	// Validate impressions (required and non-empty)
	if req.Imp == nil {
		validationErrors = append(validationErrors, models.ValidationError{
			Field:   "imp",
			Message: "At least one impression is required",
		})
	} else if len(req.Imp) == 0 {
		validationErrors = append(validationErrors, models.ValidationError{
			Field:   "imp",
			Message: "Impression array cannot be empty",
		})
	}

	// Validate site OR app (exactly one required, not both)
	hasSite := req.Site != nil
	hasApp := req.App != nil

	if !hasSite && !hasApp {
		validationErrors = append(validationErrors, models.ValidationError{
			Field:   "site/app",
			Message: "Either site or app must be present",
		})
	} else if hasSite && hasApp {
		validationErrors = append(validationErrors, models.ValidationError{
			Field:   "site/app",
			Message: "Cannot have both site and app in the same request",
		})
	}

	// Validate device (required)
	if req.Device == nil {
		validationErrors = append(validationErrors, models.ValidationError{
			Field:   "device",
			Message: "Device information is required",
		})
	}

	// If validation errors exist, return error response
	if len(validationErrors) > 0 {
		// Log validation failure
		v.logger.Warn("Bid request validation failed",
			zap.String("correlation_id", correlationID),
			zap.String("request_id", req.ID),
			zap.Int("error_count", len(validationErrors)),
		)

		// Increment error metrics
		v.metrics.ErrorsTotal.WithLabelValues("validation").Inc()
		v.metrics.RequestsTotal.WithLabelValues("invalid").Inc()

		// Return validation error response
		return models.NewValidationErrorResponse(
			"Bid request validation failed",
			models.ErrorCodeMissingID, // Use first error code
			validationErrors,
		).WithCorrelationID(correlationID)
	}

	// Log successful validation
	v.logger.Debug("Bid request validation passed",
		zap.String("correlation_id", correlationID),
		zap.String("request_id", req.ID),
		zap.Int("impression_count", len(req.Imp)),
	)

	// Increment valid request metric
	v.metrics.RequestsTotal.WithLabelValues("valid").Inc()

	return nil
}

// ValidateImpression validates individual impression object
func (v *Validator) ValidateImpression(imp *models.Impression) error {
	if imp.ID == "" {
		return fmt.Errorf("impression ID is required")
	}

	// Banner is the only supported format in this implementation
	if imp.Banner == nil {
		return fmt.Errorf("impression must have banner object (only format supported)")
	}

	return nil
}

// RecordValidationMetrics records validation metrics
func (v *Validator) RecordValidationMetrics(valid bool, duration float64) {
	if valid {
		v.metrics.LatencyHistogram.With(prometheus.Labels{
			"endpoint": "/bid",
		}).Observe(duration)
	}
}
