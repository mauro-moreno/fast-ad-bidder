package lib

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the bidder
type Metrics struct {
	// Request metrics
	RequestsTotal *prometheus.CounterVec
	BidsTotal     *prometheus.CounterVec
	WinsTotal     prometheus.Counter

	// Latency metrics
	LatencyHistogram *prometheus.HistogramVec

	// Error metrics
	ErrorsTotal *prometheus.CounterVec

	// Campaign metrics
	CampaignsActive      prometheus.Gauge
	CampaignBudgetRemaining *prometheus.GaugeVec
}

// InitMetrics initializes Prometheus metrics with the given namespace
func InitMetrics(namespace string) *Metrics {
	return &Metrics{
		// Request counters
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "requests_total",
				Help:      "Total number of bid requests received",
			},
			[]string{"status"}, // valid, invalid
		),

		BidsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "bids_total",
				Help:      "Total number of bid responses",
			},
			[]string{"result"}, // bid, nobid
		),

		WinsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "wins_total",
				Help:      "Total number of win notifications received",
			},
		),

		// Latency histogram with buckets aligned to SLA (10ms, 25ms, 50ms, 75ms, 100ms, 150ms, 200ms)
		LatencyHistogram: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "latency_seconds",
				Help:      "Bid processing latency in seconds",
				Buckets:   []float64{0.01, 0.025, 0.05, 0.075, 0.1, 0.15, 0.2},
			},
			[]string{"endpoint"},
		),

		// Error counter
		ErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors_total",
				Help:      "Total number of errors by type",
			},
			[]string{"type"}, // validation, timeout, internal
		),

		// Campaign gauges
		CampaignsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "campaigns_active",
				Help:      "Number of active campaigns loaded",
			},
		),

		CampaignBudgetRemaining: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "campaign_budget_remaining",
				Help:      "Remaining daily budget per campaign",
			},
			[]string{"campaign_id"},
		),
	}
}
