package tracker

import (
	"context"

	"github.com/fast-ad-bidder/bidder/src/models"
	"go.uber.org/zap"
)

// WinProcessor processes win notifications asynchronously
// T098: Implement async win processor goroutine
type WinProcessor struct {
	winQueue      chan *models.WinNotification
	bidCache      *BidCache
	budgetTracker *BudgetTracker
	metricsAgg    *MetricsAggregator
	influxWriter  *InfluxWriter
	logger        *zap.Logger
	workerCount   int
}

// NewWinProcessor creates a new win processor
func NewWinProcessor(
	bidCache *BidCache,
	budgetTracker *BudgetTracker,
	metricsAgg *MetricsAggregator,
	influxWriter *InfluxWriter,
	logger *zap.Logger,
	queueSize int,
	workerCount int,
) *WinProcessor {
	return &WinProcessor{
		winQueue:      make(chan *models.WinNotification, queueSize),
		bidCache:      bidCache,
		budgetTracker: budgetTracker,
		metricsAgg:    metricsAgg,
		influxWriter:  influxWriter,
		logger:        logger,
		workerCount:   workerCount,
	}
}

// Start starts the win processing workers
func (wp *WinProcessor) Start(ctx context.Context) {
	for i := 0; i < wp.workerCount; i++ {
		go wp.worker(ctx, i)
	}

	wp.logger.Info("Win processor started",
		zap.Int("worker_count", wp.workerCount),
		zap.Int("queue_size", cap(wp.winQueue)),
	)
}

// QueueWin queues a win notification for processing
func (wp *WinProcessor) QueueWin(win *models.WinNotification) {
	select {
	case wp.winQueue <- win:
		wp.logger.Debug("Win notification queued",
			zap.String("bid_id", win.BidID),
			zap.Float64("price", win.Price),
		)
	default:
		wp.logger.Warn("Win queue full, dropping win notification",
			zap.String("bid_id", win.BidID),
		)
	}
}

// worker processes win notifications from the queue
func (wp *WinProcessor) worker(ctx context.Context, workerID int) {
	wp.logger.Info("Win processor worker started",
		zap.Int("worker_id", workerID),
	)

	for {
		select {
		case win := <-wp.winQueue:
			wp.processWin(ctx, win, workerID)
		case <-ctx.Done():
			wp.logger.Info("Win processor worker stopped",
				zap.Int("worker_id", workerID),
			)
			return
		}
	}
}

// processWin processes a single win notification
// T101: Add structured logging for win notifications
func (wp *WinProcessor) processWin(ctx context.Context, win *models.WinNotification, workerID int) {
	// Look up the bid in cache
	cachedBid, found := wp.bidCache.Get(win.BidID)
	if !found {
		wp.logger.Warn("Win notification for unknown or expired bid",
			zap.String("bid_id", win.BidID),
			zap.Float64("price", win.Price),
			zap.Int("worker_id", workerID),
		)
		return
	}

	wp.logger.Info("Processing win notification",
		zap.String("bid_id", win.BidID),
		zap.String("campaign_id", cachedBid.CampaignID),
		zap.Float64("win_price", win.Price),
		zap.Float64("bid_price", cachedBid.Price),
		zap.Int("worker_id", workerID),
	)

	// Deduct budget
	if err := wp.budgetTracker.DeductBudget(ctx, cachedBid.CampaignID, win.Price); err != nil {
		wp.logger.Error("Failed to deduct budget for win",
			zap.String("bid_id", win.BidID),
			zap.String("campaign_id", cachedBid.CampaignID),
			zap.Float64("price", win.Price),
			zap.Error(err),
		)
	}

	// Record win in metrics
	wp.metricsAgg.RecordWin(cachedBid.CampaignID, win.Price)

	// Write to InfluxDB
	winEvent := &models.WinEvent{
		CampaignID: cachedBid.CampaignID,
		BidID:      win.BidID,
		WinPrice:   win.Price,
		Timestamp:  win.Timestamp,
	}

	if err := wp.influxWriter.WriteWinEvent(ctx, winEvent); err != nil {
		wp.logger.Error("Failed to write win event to InfluxDB",
			zap.String("bid_id", win.BidID),
			zap.String("campaign_id", cachedBid.CampaignID),
			zap.Error(err),
		)
	}

	// Remove bid from cache after processing
	wp.bidCache.Delete(win.BidID)

	wp.logger.Info("Win notification processed successfully",
		zap.String("bid_id", win.BidID),
		zap.String("campaign_id", cachedBid.CampaignID),
		zap.Float64("win_price", win.Price),
	)
}

// Close gracefully shuts down the win processor
func (wp *WinProcessor) Close(ctx context.Context) error {
	close(wp.winQueue)

	// Flush any pending InfluxDB writes
	if err := wp.influxWriter.Flush(ctx); err != nil {
		return err
	}

	return nil
}
