package matcher

import (
	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/store"
)

// DomainMatcher matches campaigns based on site/app domain targeting
type DomainMatcher struct{}

// NewDomainMatcher creates a new domain matcher
func NewDomainMatcher() *DomainMatcher {
	return &DomainMatcher{}
}

// Matches returns true if the site or app matches campaign targeting
func (m *DomainMatcher) Matches(campaign *store.Campaign, site *models.Site, app *models.App) bool {
	// Check site domain targeting
	if site != nil {
		return m.matchesSiteDomain(campaign, site.Domain)
	}

	// Check app bundle targeting
	if app != nil {
		return m.matchesAppBundle(campaign, app.Bundle)
	}

	// No site or app provided
	return false
}

// matchesSiteDomain checks if the site domain matches campaign targeting
func (m *DomainMatcher) matchesSiteDomain(campaign *store.Campaign, siteDomain string) bool {
	// No domain targeting means match all
	domainsTargeting, hasDomains := campaign.Targeting["domains"]
	if !hasDomains {
		return true
	}

	domainsMap, ok := domainsTargeting.(map[string]interface{})
	if !ok {
		return true // Invalid targeting format, default to match
	}

	// Check whitelist
	if whitelist, hasWhitelist := domainsMap["whitelist"]; hasWhitelist {
		whitelistArr, ok := whitelist.([]interface{})
		if ok {
			return matchesDomainInList(siteDomain, whitelistArr)
		}
	}

	// No whitelist means match all
	return true
}

// matchesAppBundle checks if the app bundle matches campaign targeting
func (m *DomainMatcher) matchesAppBundle(campaign *store.Campaign, appBundle string) bool {
	// No app bundle targeting means match all
	appBundlesTargeting, hasAppBundles := campaign.Targeting["app_bundles"]
	if !hasAppBundles {
		return true
	}

	appBundlesList, ok := appBundlesTargeting.([]interface{})
	if !ok {
		return true // Invalid targeting format, default to match
	}

	return matchesDomainInList(appBundle, appBundlesList)
}

// matchesDomainInList checks if the domain is in the target list
func matchesDomainInList(domain string, targetList []interface{}) bool {
	if domain == "" {
		return false
	}

	for _, target := range targetList {
		if targetStr, ok := target.(string); ok {
			if targetStr == domain {
				return true
			}
		}
	}

	return false
}
