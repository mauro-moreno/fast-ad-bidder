package matcher

import (
	"testing"

	"github.com/fast-ad-bidder/bidder/src/models"
	"github.com/fast-ad-bidder/bidder/src/services/matcher"
	"github.com/fast-ad-bidder/bidder/src/services/store"
	"github.com/stretchr/testify/assert"
)

// int64Ptr is a helper to create int64 pointers
func int64Ptr(v int64) *int64 {
	return &v
}

// T057: Unit test - creative dimension matcher
func TestCreativeDimensionMatcher(t *testing.T) {
	creativeMatcher := matcher.NewCreativeMatcher()

	tests := []struct {
		name        string
		creative    *store.Creative
		impression  *models.Impression
		shouldMatch bool
	}{
		{
			name: "Match - Exact dimensions",
			creative: &store.Creative{
				ID:     "creative-001",
				Width:  300,
				Height: 250,
			},
			impression: &models.Impression{
				ID: "imp-1",
				Banner: &models.Banner{
					W: int64Ptr(300),
					H: int64Ptr(250),
				},
			},
			shouldMatch: true,
		},
		{
			name: "No Match - Width mismatch",
			creative: &store.Creative{
				ID:     "creative-001",
				Width:  300,
				Height: 250,
			},
			impression: &models.Impression{
				ID: "imp-1",
				Banner: &models.Banner{
					W: int64Ptr(728),
					H: int64Ptr(250),
				},
			},
			shouldMatch: false,
		},
		{
			name: "No Match - Height mismatch",
			creative: &store.Creative{
				ID:     "creative-001",
				Width:  300,
				Height: 250,
			},
			impression: &models.Impression{
				ID: "imp-1",
				Banner: &models.Banner{
					W: int64Ptr(300),
					H: int64Ptr(600),
				},
			},
			shouldMatch: false,
		},
		{
			name: "Match - Leaderboard dimensions",
			creative: &store.Creative{
				ID:     "creative-002",
				Width:  728,
				Height: 90,
			},
			impression: &models.Impression{
				ID: "imp-2",
				Banner: &models.Banner{
					W: int64Ptr(728),
					H: int64Ptr(90),
				},
			},
			shouldMatch: true,
		},
		{
			name: "Match - Mobile banner dimensions",
			creative: &store.Creative{
				ID:     "creative-003",
				Width:  320,
				Height: 50,
			},
			impression: &models.Impression{
				ID: "imp-3",
				Banner: &models.Banner{
					W: int64Ptr(320),
					H: int64Ptr(50),
				},
			},
			shouldMatch: true,
		},
		{
			name: "No Match - No banner in impression",
			creative: &store.Creative{
				ID:     "creative-001",
				Width:  300,
				Height: 250,
			},
			impression: &models.Impression{
				ID: "imp-1",
				// No Banner field
			},
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := creativeMatcher.Matches(tt.creative, tt.impression)
			assert.Equal(t, tt.shouldMatch, matches, "Creative dimension match result mismatch")
		})
	}
}
