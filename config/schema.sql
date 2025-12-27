-- OpenRTB Bidder Database Schema
-- PostgreSQL 14+

-- Campaigns table
CREATE TABLE IF NOT EXISTS campaigns (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    advertiser_id   TEXT NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('active', 'paused', 'budget_capped')),
    daily_budget    DECIMAL(10,2) NOT NULL CHECK (daily_budget > 0),
    spent_today     DECIMAL(10,2) NOT NULL DEFAULT 0 CHECK (spent_today >= 0),
    budget_reset_time TIMESTAMPTZ NOT NULL,

    -- Targeting rules stored as JSONB
    targeting       JSONB NOT NULL DEFAULT '{}',

    -- Bid strategy
    bid_strategy    TEXT NOT NULL CHECK (bid_strategy IN ('fixed_cpm', 'dynamic')),
    fixed_cpm       DECIMAL(10,2) CHECK (fixed_cpm IS NULL OR fixed_cpm > 0),
    max_cpm         DECIMAL(10,2) CHECK (max_cpm IS NULL OR max_cpm > 0),

    -- Creative assignment
    creative_ids    TEXT[] NOT NULL DEFAULT '{}',

    -- Metadata
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for campaigns
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status);
CREATE INDEX IF NOT EXISTS idx_campaigns_advertiser ON campaigns(advertiser_id);
CREATE INDEX IF NOT EXISTS idx_campaigns_budget_reset ON campaigns(budget_reset_time);

-- Creatives table
CREATE TABLE IF NOT EXISTS creatives (
    id                  TEXT PRIMARY KEY,
    campaign_id         TEXT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    width               INT NOT NULL CHECK (width > 0),
    height              INT NOT NULL CHECK (height > 0),
    format              TEXT NOT NULL DEFAULT 'banner' CHECK (format IN ('banner', 'video', 'native')),
    markup              TEXT NOT NULL,
    click_through_url   TEXT NOT NULL,
    advertiser_domain   TEXT NOT NULL,
    approval_status     TEXT NOT NULL CHECK (approval_status IN ('approved', 'pending', 'rejected')),
    exchange_approvals  JSONB NOT NULL DEFAULT '{}',

    -- Tracking
    impression_trackers TEXT[] NOT NULL DEFAULT '{}',
    click_trackers      TEXT[] NOT NULL DEFAULT '{}',

    -- Metadata
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for creatives
CREATE INDEX IF NOT EXISTS idx_creatives_campaign ON creatives(campaign_id);
CREATE INDEX IF NOT EXISTS idx_creatives_dimensions ON creatives(width, height);
CREATE INDEX IF NOT EXISTS idx_creatives_approval ON creatives(approval_status);
CREATE INDEX IF NOT EXISTS idx_creatives_format ON creatives(format);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_campaigns_updated_at BEFORE UPDATE ON campaigns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_creatives_updated_at BEFORE UPDATE ON creatives
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
