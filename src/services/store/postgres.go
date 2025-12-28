package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
)

// PostgresLoader implements CampaignLoader for PostgreSQL
type PostgresLoader struct {
	db *sql.DB
}

// NewPostgresLoader creates a new PostgreSQL campaign loader
func NewPostgresLoader(db *sql.DB) *PostgresLoader {
	return &PostgresLoader{
		db: db,
	}
}

// LoadFromDB loads campaigns and creatives from PostgreSQL
func (l *PostgresLoader) LoadFromDB(ctx context.Context) ([]*Campaign, []*Creative, error) {
	campaigns, err := l.loadCampaigns(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load campaigns: %w", err)
	}

	creatives, err := l.loadCreatives(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load creatives: %w", err)
	}

	return campaigns, creatives, nil
}

// loadCampaigns loads all active campaigns from the database
func (l *PostgresLoader) loadCampaigns(ctx context.Context) ([]*Campaign, error) {
	query := `
		SELECT
			id,
			name,
			status,
			daily_budget,
			spent_today,
			targeting,
			bid_strategy,
			creative_ids,
			fixed_cpm,
			max_cpm
		FROM campaigns
		WHERE status IN ('active', 'paused')
		ORDER BY created_at DESC
	`

	rows, err := l.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query campaigns: %w", err)
	}
	defer rows.Close()

	var campaigns []*Campaign
	for rows.Next() {
		var campaign Campaign
		var targetingJSON []byte
		var fixedCPM sql.NullFloat64
		var maxCPM sql.NullFloat64

		err := rows.Scan(
			&campaign.ID,
			&campaign.Name,
			&campaign.Status,
			&campaign.DailyBudget,
			&campaign.SpentToday,
			&targetingJSON,
			&campaign.BidStrategy,
			pq.Array(&campaign.CreativeIDs),
			&fixedCPM,
			&maxCPM,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan campaign: %w", err)
		}

		// Handle nullable float fields
		if fixedCPM.Valid {
			campaign.BidFloorCPM = fixedCPM.Float64
		}
		if maxCPM.Valid {
			campaign.MaxBidCPM = maxCPM.Float64
		}

		// Parse targeting JSON
		if err := json.Unmarshal(targetingJSON, &campaign.Targeting); err != nil {
			return nil, fmt.Errorf("failed to parse targeting for campaign %s: %w", campaign.ID, err)
		}

		campaigns = append(campaigns, &campaign)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating campaigns: %w", err)
	}

	return campaigns, nil
}

// loadCreatives loads all approved creatives from the database
func (l *PostgresLoader) loadCreatives(ctx context.Context) ([]*Creative, error) {
	query := `
		SELECT
			id,
			campaign_id,
			width,
			height,
			markup,
			approval_status,
			exchange_approvals
		FROM creatives
		WHERE approval_status = 'approved'
		ORDER BY created_at DESC
	`

	rows, err := l.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query creatives: %w", err)
	}
	defer rows.Close()

	var creatives []*Creative
	for rows.Next() {
		var creative Creative
		var exchangeApprovalsJSON []byte

		err := rows.Scan(
			&creative.ID,
			&creative.CampaignID,
			&creative.Width,
			&creative.Height,
			&creative.Markup,
			&creative.ApprovalStatus,
			&exchangeApprovalsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan creative: %w", err)
		}

		// Parse exchange approvals JSON
		if err := json.Unmarshal(exchangeApprovalsJSON, &creative.ExchangeApprovals); err != nil {
			return nil, fmt.Errorf("failed to parse exchange_approvals for creative %s: %w", creative.ID, err)
		}

		creatives = append(creatives, &creative)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating creatives: %w", err)
	}

	return creatives, nil
}

// ConnectPostgres creates a new PostgreSQL connection
func ConnectPostgres(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0) // Connections are reused forever

	return db, nil
}
