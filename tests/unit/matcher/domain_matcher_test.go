package matcher

import (
	"testing"

	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/matcher"
	"github.com/fast-ad-bidder/bidder/src/services/store"
	"github.com/stretchr/testify/assert"
)

// T056: Unit test - campaign site/app domain matcher
func TestDomainMatcher(t *testing.T) {
	domainMatcher := matcher.NewDomainMatcher()

	tests := []struct {
		name        string
		campaign    *store.Campaign
		site        *models.Site
		app         *models.App
		shouldMatch bool
	}{
		{
			name: "Match - Site domain in whitelist",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"domains": map[string]interface{}{
						"whitelist": []interface{}{"example.com", "test.com"},
					},
				},
			},
			site: &models.Site{
				Domain: "example.com",
			},
			shouldMatch: true,
		},
		{
			name: "No Match - Site domain not in whitelist",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"domains": map[string]interface{}{
						"whitelist": []interface{}{"example.com", "test.com"},
					},
				},
			},
			site: &models.Site{
				Domain: "other.com",
			},
			shouldMatch: false,
		},
		{
			name: "Match - App bundle in whitelist",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"app_bundles": []interface{}{"com.example.app", "com.test.app"},
				},
			},
			app: &models.App{
				Bundle: "com.example.app",
			},
			shouldMatch: true,
		},
		{
			name: "No Match - App bundle not in whitelist",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"app_bundles": []interface{}{"com.example.app"},
				},
			},
			app: &models.App{
				Bundle: "com.other.app",
			},
			shouldMatch: false,
		},
		{
			name: "Match - No domain targeting (matches all)",
			campaign: &store.Campaign{
				ID:        "test-campaign",
				Targeting: map[string]interface{}{},
			},
			site: &models.Site{
				Domain: "any-domain.com",
			},
			shouldMatch: true,
		},
		{
			name: "Match - Subdomain matching",
			campaign: &store.Campaign{
				ID: "test-campaign",
				Targeting: map[string]interface{}{
					"domains": map[string]interface{}{
						"whitelist": []interface{}{"example.com", "news.example.com"},
					},
				},
			},
			site: &models.Site{
				Domain: "news.example.com",
			},
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := domainMatcher.Matches(tt.campaign, tt.site, tt.app)
			assert.Equal(t, tt.shouldMatch, matches, "Domain match result mismatch")
		})
	}
}
