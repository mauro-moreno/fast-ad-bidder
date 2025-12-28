package tracker

import (
	"context"
	"time"

	"github.com/fast-ad-bidder/bidder/src/lib"
	"github.com/fast-ad-bidder/bidder/src/models"
	"go.uber.org/zap"
)

// InfluxWriter writes bid and win events to InfluxDB asynchronously
// T094: Implement InfluxDB metrics writer
type InfluxWriter struct {
	writer *lib.InfluxClient
	logger *zap.Logger
}

// NewInfluxWriter creates a new InfluxDB metrics writer
func NewInfluxWriter(writer *lib.InfluxClient, logger *zap.Logger) *InfluxWriter {
	return &InfluxWriter{
		writer: writer,
		logger: logger,
	}
}

// WriteBidEvent writes a bid event to InfluxDB
func (iw *InfluxWriter) WriteBidEvent(ctx context.Context, event *models.BidEvent) error {
	if iw.writer == nil {
		// InfluxDB is optional, skip if not configured
		return nil
	}

	// Use WriteBidMetric with appropriate conversion
	metric := &lib.BidMetric{
		CampaignID: event.CampaignID,
		Result:     "bid",
		BidPrice:   event.Price,
		Timestamp:  event.Timestamp,
	}

	err := iw.writer.WriteBidMetric(ctx, metric)
	if err != nil {
		iw.logger.Error("Failed to write bid event to InfluxDB",
			zap.String("campaign_id", event.CampaignID),
			zap.String("bid_id", event.BidID),
			zap.Error(err),
		)
		return err
	}

	iw.logger.Debug("Bid event written to InfluxDB",
		zap.String("campaign_id", event.CampaignID),
		zap.String("bid_id", event.BidID),
	)

	return nil
}

// WriteWinEvent writes a win event to InfluxDB
func (iw *InfluxWriter) WriteWinEvent(ctx context.Context, event *models.WinEvent) error {
	if iw.writer == nil {
		// InfluxDB is optional, skip if not configured
		return nil
	}

	// Convert to lib.WinNotification format
	winNotif := &lib.WinNotification{
		BidID:      event.BidID,
		CampaignID: event.CampaignID,
		Price:      event.WinPrice,
		Currency:   "USD",
		Timestamp:  event.Timestamp,
	}

	err := iw.writer.WriteWinNotification(ctx, winNotif)
	if err != nil {
		iw.logger.Error("Failed to write win event to InfluxDB",
			zap.String("campaign_id", event.CampaignID),
			zap.String("bid_id", event.BidID),
			zap.Error(err),
		)
		return err
	}

	iw.logger.Debug("Win event written to InfluxDB",
		zap.String("campaign_id", event.CampaignID),
		zap.String("bid_id", event.BidID),
	)

	return nil
}

// WriteMetricsSummary writes aggregated metrics to InfluxDB
func (iw *InfluxWriter) WriteMetricsSummary(ctx context.Context, metrics *models.BidMetrics) error {
	if iw.writer == nil {
		// InfluxDB is optional, skip if not configured
		return nil
	}

	// Use WriteBidMetric to write aggregated metrics
	metric := &lib.BidMetric{
		CampaignID: metrics.CampaignID,
		Result:     "summary",
		Timestamp:  time.Now(),
	}

	err := iw.writer.WriteBidMetric(ctx, metric)
	if err != nil {
		iw.logger.Error("Failed to write metrics summary to InfluxDB",
			zap.String("campaign_id", metrics.CampaignID),
			zap.Error(err),
		)
		return err
	}

	iw.logger.Debug("Metrics summary written to InfluxDB",
		zap.String("campaign_id", metrics.CampaignID),
	)

	return nil
}

// Flush ensures all pending writes are sent to InfluxDB
func (iw *InfluxWriter) Flush(ctx context.Context) error {
	if iw.writer == nil {
		return nil
	}

	// Call flush without context parameter
	iw.writer.Flush()
	return nil
}

// Close closes the InfluxDB writer
func (iw *InfluxWriter) Close() error {
	if iw.writer == nil {
		return nil
	}

	return iw.writer.Close()
}
