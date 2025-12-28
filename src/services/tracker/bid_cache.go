package tracker

import (
	"sync"
	"time"
)

// CachedBid represents a bid stored in the cache
type CachedBid struct {
	BidID      string
	CampaignID string
	Price      float64
	Timestamp  time.Time
}

// BidCache implements a thread-safe bid cache with TTL
// T090: Implement bid cache with 5-minute TTL
type BidCache struct {
	cache map[string]*CachedBid
	mu    sync.RWMutex
	ttl   time.Duration
}

// NewBidCache creates a new bid cache with the specified TTL
func NewBidCache(ttl time.Duration) *BidCache {
	cache := &BidCache{
		cache: make(map[string]*CachedBid),
		ttl:   ttl,
	}

	// Start cleanup goroutine
	go cache.cleanupLoop()

	return cache
}

// Store adds a bid to the cache
func (bc *BidCache) Store(bidID string, bid *CachedBid) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.cache[bidID] = bid
}

// Get retrieves a bid from the cache
// Returns the bid and true if found and not expired, nil and false otherwise
func (bc *BidCache) Get(bidID string) (*CachedBid, bool) {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	bid, exists := bc.cache[bidID]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Since(bid.Timestamp) > bc.ttl {
		return nil, false
	}

	return bid, true
}

// Delete removes a bid from the cache
func (bc *BidCache) Delete(bidID string) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	delete(bc.cache, bidID)
}

// Cleanup removes all expired entries from the cache
func (bc *BidCache) Cleanup() {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	now := time.Now()
	for bidID, bid := range bc.cache {
		if now.Sub(bid.Timestamp) > bc.ttl {
			delete(bc.cache, bidID)
		}
	}
}

// cleanupLoop runs periodic cleanup of expired entries
func (bc *BidCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		bc.Cleanup()
	}
}

// Size returns the current number of entries in the cache
func (bc *BidCache) Size() int {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	return len(bc.cache)
}
