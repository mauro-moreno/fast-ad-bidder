package tracker

import (
	"testing"
	"time"

	"github.com/fast-ad-bidder/bidder/src/services/tracker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T081: Unit test - bid cache lookup and expiration (5min TTL)
func TestBidCacheLookupAndExpiration(t *testing.T) {
	cache := tracker.NewBidCache(5 * time.Minute)

	t.Run("Store and retrieve bid", func(t *testing.T) {
		bidID := "test-bid-123"
		bidInfo := &tracker.CachedBid{
			BidID:      bidID,
			CampaignID: "campaign-001",
			Price:      5.25,
			Timestamp:  time.Now(),
		}

		// Store bid
		cache.Store(bidID, bidInfo)

		// Retrieve bid
		retrieved, found := cache.Get(bidID)
		require.True(t, found, "Bid should be found in cache")
		assert.Equal(t, bidInfo.BidID, retrieved.BidID)
		assert.Equal(t, bidInfo.CampaignID, retrieved.CampaignID)
		assert.Equal(t, bidInfo.Price, retrieved.Price)
	})

	t.Run("Bid not found", func(t *testing.T) {
		_, found := cache.Get("nonexistent-bid")
		assert.False(t, found, "Nonexistent bid should not be found")
	})

	t.Run("Bid expiration after TTL", func(t *testing.T) {
		// Use very short TTL for testing
		shortCache := tracker.NewBidCache(100 * time.Millisecond)

		bidID := "expiring-bid-123"
		bidInfo := &tracker.CachedBid{
			BidID:      bidID,
			CampaignID: "campaign-001",
			Price:      5.25,
			Timestamp:  time.Now(),
		}

		shortCache.Store(bidID, bidInfo)

		// Should be found immediately
		_, found := shortCache.Get(bidID)
		assert.True(t, found, "Bid should be found immediately after storing")

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Should be expired
		_, found = shortCache.Get(bidID)
		assert.False(t, found, "Bid should be expired after TTL")
	})

	t.Run("Update existing bid", func(t *testing.T) {
		bidID := "update-bid-123"
		bidInfo1 := &tracker.CachedBid{
			BidID:      bidID,
			CampaignID: "campaign-001",
			Price:      5.25,
			Timestamp:  time.Now(),
		}

		cache.Store(bidID, bidInfo1)

		// Update with new price
		bidInfo2 := &tracker.CachedBid{
			BidID:      bidID,
			CampaignID: "campaign-001",
			Price:      6.50,
			Timestamp:  time.Now(),
		}

		cache.Store(bidID, bidInfo2)

		// Should have updated price
		retrieved, found := cache.Get(bidID)
		require.True(t, found)
		assert.Equal(t, 6.50, retrieved.Price)
	})

	t.Run("Delete bid from cache", func(t *testing.T) {
		bidID := "delete-bid-123"
		bidInfo := &tracker.CachedBid{
			BidID:      bidID,
			CampaignID: "campaign-001",
			Price:      5.25,
			Timestamp:  time.Now(),
		}

		cache.Store(bidID, bidInfo)

		// Delete bid
		cache.Delete(bidID)

		// Should not be found
		_, found := cache.Get(bidID)
		assert.False(t, found, "Deleted bid should not be found")
	})

	t.Run("Cache cleanup removes expired entries", func(t *testing.T) {
		shortCache := tracker.NewBidCache(100 * time.Millisecond)

		// Store multiple bids
		for i := 0; i < 5; i++ {
			bidInfo := &tracker.CachedBid{
				BidID:      string(rune('A' + i)),
				CampaignID: "campaign-001",
				Price:      5.25,
				Timestamp:  time.Now(),
			}
			shortCache.Store(bidInfo.BidID, bidInfo)
		}

		// Wait for expiration
		time.Sleep(150 * time.Millisecond)

		// Trigger cleanup (implementation should have periodic cleanup)
		shortCache.Cleanup()

		// All should be expired
		for i := 0; i < 5; i++ {
			_, found := shortCache.Get(string(rune('A' + i)))
			assert.False(t, found, "All expired bids should be cleaned up")
		}
	})
}

// Test concurrent access to bid cache
func TestBidCacheConcurrentAccess(t *testing.T) {
	cache := tracker.NewBidCache(5 * time.Minute)

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			bidInfo := &tracker.CachedBid{
				BidID:      string(rune('A' + id)),
				CampaignID: "campaign-001",
				Price:      5.25,
				Timestamp:  time.Now(),
			}
			cache.Store(bidInfo.BidID, bidInfo)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func(id int) {
			_, _ = cache.Get(string(rune('A' + id)))
			done <- true
		}(i)
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}

	// No race conditions should occur
}
