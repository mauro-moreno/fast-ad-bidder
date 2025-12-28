-- Sample campaign and creative data for local development

-- Sample Campaign 1: US Mobile Banner
INSERT INTO campaigns (
    id, name, advertiser_id, status, daily_budget, spent_today, budget_reset_time,
    targeting, bid_strategy, fixed_cpm, creative_ids
) VALUES (
    'campaign-001',
    'US Mobile Banner Campaign',
    'advertiser-001',
    'active',
    100.00,
    0.00,
    NOW() + INTERVAL '1 day',
    '{"GeoTargeting": ["US"], "DeviceTypes": ["mobile"], "SiteDomains": [], "AppBundles": [], "MinViewability": 0.0}'::jsonb,
    'fixed_cpm',
    2.50,
    ARRAY['creative-001', 'creative-002']
);

-- Sample Campaign 2: Global Desktop Banner
INSERT INTO campaigns (
    id, name, advertiser_id, status, daily_budget, spent_today, budget_reset_time,
    targeting, bid_strategy, fixed_cpm, creative_ids
) VALUES (
    'campaign-002',
    'Global Desktop Banner Campaign',
    'advertiser-002',
    'active',
    200.00,
    0.00,
    NOW() + INTERVAL '1 day',
    '{"GeoTargeting": ["US", "CA", "GB"], "DeviceTypes": ["desktop"], "SiteDomains": [], "AppBundles": [], "MinViewability": 0.5}'::jsonb,
    'fixed_cpm',
    3.00,
    ARRAY['creative-003']
);

-- Sample Campaign 3: Paused Campaign (for testing)
INSERT INTO campaigns (
    id, name, advertiser_id, status, daily_budget, spent_today, budget_reset_time,
    targeting, bid_strategy, fixed_cpm, creative_ids
) VALUES (
    'campaign-003',
    'Paused Test Campaign',
    'advertiser-001',
    'paused',
    50.00,
    0.00,
    NOW() + INTERVAL '1 day',
    '{"GeoTargeting": ["US"], "DeviceTypes": ["mobile", "desktop"], "SiteDomains": [], "AppBundles": [], "MinViewability": 0.0}'::jsonb,
    'fixed_cpm',
    1.50,
    ARRAY['creative-004']
);

-- Sample Creative 1: Mobile Banner 300x250
INSERT INTO creatives (
    id, campaign_id, name, width, height, format, markup,
    click_through_url, advertiser_domain, approval_status, exchange_approvals,
    impression_trackers, click_trackers
) VALUES (
    'creative-001',
    'campaign-001',
    'Mobile Banner 300x250',
    300,
    250,
    'banner',
    '<div style="width:300px;height:250px;background:#4285f4;display:flex;align-items:center;justify-content:center;color:white;font-size:24px;">Sample Ad</div>',
    'https://advertiser.example.com/landing',
    'advertiser.example.com',
    'approved',
    '{"google_adx": true}'::jsonb,
    ARRAY['https://tracker.example.com/imp?id=creative-001'],
    ARRAY['https://tracker.example.com/click?id=creative-001']
);

-- Sample Creative 2: Mobile Banner 320x50
INSERT INTO creatives (
    id, campaign_id, name, width, height, format, markup,
    click_through_url, advertiser_domain, approval_status, exchange_approvals,
    impression_trackers, click_trackers
) VALUES (
    'creative-002',
    'campaign-001',
    'Mobile Banner 320x50',
    320,
    50,
    'banner',
    '<div style="width:320px;height:50px;background:#ea4335;display:flex;align-items:center;justify-content:center;color:white;font-size:16px;">Ad Banner</div>',
    'https://advertiser.example.com/mobile',
    'advertiser.example.com',
    'approved',
    '{"google_adx": true}'::jsonb,
    ARRAY['https://tracker.example.com/imp?id=creative-002'],
    ARRAY['https://tracker.example.com/click?id=creative-002']
);

-- Sample Creative 3: Desktop Banner 728x90
INSERT INTO creatives (
    id, campaign_id, name, width, height, format, markup,
    click_through_url, advertiser_domain, approval_status, exchange_approvals,
    impression_trackers, click_trackers
) VALUES (
    'creative-003',
    'campaign-002',
    'Desktop Leaderboard 728x90',
    728,
    90,
    'banner',
    '<div style="width:728px;height:90px;background:#34a853;display:flex;align-items:center;justify-content:center;color:white;font-size:20px;">Leaderboard Ad</div>',
    'https://advertiser2.example.com/promo',
    'advertiser2.example.com',
    'approved',
    '{"google_adx": true}'::jsonb,
    ARRAY['https://tracker.example.com/imp?id=creative-003'],
    ARRAY['https://tracker.example.com/click?id=creative-003']
);

-- Sample Creative 4: Pending approval (for testing)
INSERT INTO creatives (
    id, campaign_id, name, width, height, format, markup,
    click_through_url, advertiser_domain, approval_status, exchange_approvals,
    impression_trackers, click_trackers
) VALUES (
    'creative-004',
    'campaign-003',
    'Pending Creative 300x250',
    300,
    250,
    'banner',
    '<div style="width:300px;height:250px;background:#fbbc04;display:flex;align-items:center;justify-content:center;color:white;">Pending</div>',
    'https://advertiser.example.com/test',
    'advertiser.example.com',
    'pending',
    '{}'::jsonb,
    ARRAY[]::text[],
    ARRAY[]::text[]
);
