package lib

import (
	"context"
	"fmt"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

// InfluxClient wraps the InfluxDB client for time-series metrics
type InfluxClient struct {
	client   influxdb2.Client
	writeAPI api.WriteAPI
	bucket   string
	org      string
}

// InfluxConfig holds InfluxDB connection configuration
type InfluxConfig struct {
	URL    string
	Token  string
	Org    string
	Bucket string
}

// NewInfluxClient creates a new InfluxDB client wrapper
func NewInfluxClient(cfg InfluxConfig) *InfluxClient {
	client := influxdb2.NewClient(cfg.URL, cfg.Token)
	writeAPI := client.WriteAPI(cfg.Org, cfg.Bucket)

	return &InfluxClient{
		client:   client,
		writeAPI: writeAPI,
		bucket:   cfg.Bucket,
		org:      cfg.Org,
	}
}

// WinNotification represents a win notification event
type WinNotification struct {
	BidID        string
	CampaignID   string
	CreativeID   string
	Price        float64
	Currency     string
	ImpressionID string
	Timestamp    time.Time
}

// WriteWinNotification writes a win notification to InfluxDB
func (c *InfluxClient) WriteWinNotification(ctx context.Context, win *WinNotification) error {
	point := influxdb2.NewPoint(
		"win_notifications",
		map[string]string{
			"bid_id":      win.BidID,
			"campaign_id": win.CampaignID,
			"creative_id": win.CreativeID,
			"currency":    win.Currency,
		},
		map[string]interface{}{
			"price":         win.Price,
			"impression_id": win.ImpressionID,
		},
		win.Timestamp,
	)

	c.writeAPI.WritePoint(point)

	return nil
}

// BidMetric represents a bid processing metric
type BidMetric struct {
	CampaignID   string
	Result       string // "bid" or "nobid"
	Latency      float64
	BidPrice     float64
	BidFloorCPM  float64
	Timestamp    time.Time
}

// WriteBidMetric writes a bid processing metric to InfluxDB
func (c *InfluxClient) WriteBidMetric(ctx context.Context, metric *BidMetric) error {
	point := influxdb2.NewPoint(
		"bid_metrics",
		map[string]string{
			"campaign_id": metric.CampaignID,
			"result":      metric.Result,
		},
		map[string]interface{}{
			"latency":       metric.Latency,
			"bid_price":     metric.BidPrice,
			"bid_floor_cpm": metric.BidFloorCPM,
		},
		metric.Timestamp,
	)

	c.writeAPI.WritePoint(point)

	return nil
}

// Flush forces all pending writes to complete
func (c *InfluxClient) Flush() {
	c.writeAPI.Flush()
}

// Close closes the InfluxDB client and flushes pending writes
func (c *InfluxClient) Close() error {
	c.writeAPI.Flush()
	c.client.Close()
	return nil
}

// HealthCheck verifies the InfluxDB connection is healthy
func (c *InfluxClient) HealthCheck(ctx context.Context) error {
	health, err := c.client.Health(ctx)
	if err != nil {
		return fmt.Errorf("influxdb health check failed: %w", err)
	}

	if health.Status != "pass" {
		return fmt.Errorf("influxdb health status: %s", health.Status)
	}

	return nil
}
