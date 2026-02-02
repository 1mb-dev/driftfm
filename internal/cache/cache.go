package cache

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Default cache configuration
const (
	DefaultTTL      = 5 * time.Minute // Playlist refresh interval
	CleanupInterval = 1 * time.Minute // Expired entry cleanup
)

// Cache keys
const (
	KeyMoodsList = "moods:list"
	KeyPlaylist  = "playlist:%s" // playlist:{mood}
)

type entry struct {
	value     any
	expiresAt time.Time
}

// Cache is a simple in-memory key-value store with TTL expiration.
type Cache struct {
	mu      sync.RWMutex
	items   map[string]entry
	hits    atomic.Int64
	misses  atomic.Int64
	stopCh  chan struct{}
	stopped chan struct{}
}

// New creates a new cache that periodically evicts expired entries.
func New() (*Cache, error) {
	c := &Cache{
		items:   make(map[string]entry),
		stopCh:  make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go c.cleanup()
	return c, nil
}

func (c *Cache) cleanup() {
	defer close(c.stopped)
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.evictExpired()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Cache) evictExpired() {
	now := time.Now()
	c.mu.Lock()
	for k, e := range c.items {
		if now.After(e.expiresAt) {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}

// Get retrieves a value from cache. Returns (nil, false) on miss or expiry.
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expiresAt) {
		c.misses.Add(1)
		return nil, false
	}
	c.hits.Add(1)
	return e.value, true
}

// Set stores a value with the default TTL.
func (c *Cache) Set(key string, value any) error {
	c.mu.Lock()
	c.items[key] = entry{value: value, expiresAt: time.Now().Add(DefaultTTL)}
	c.mu.Unlock()
	return nil
}

// PlaylistKey returns the cache key for a mood's playlist.
func PlaylistKey(mood string) string {
	return fmt.Sprintf(KeyPlaylist, mood)
}

// Stats returns cache statistics for the metrics endpoint.
func (c *Cache) Stats() map[string]any {
	hits := c.hits.Load()
	misses := c.misses.Load()
	total := hits + misses
	var hitRate float64
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}
	c.mu.RLock()
	keyCount := len(c.items)
	c.mu.RUnlock()
	return map[string]any{
		"hits":      hits,
		"misses":    misses,
		"hit_rate":  hitRate,
		"key_count": keyCount,
		"total":     total,
	}
}

// InvalidateMoods clears all mood-related cache entries.
func (c *Cache) InvalidateMoods() {
	c.mu.Lock()
	delete(c.items, KeyMoodsList)
	for k := range c.items {
		if strings.HasPrefix(k, "playlist:") {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}

// Close stops the cleanup goroutine.
func (c *Cache) Close() error {
	close(c.stopCh)
	<-c.stopped
	return nil
}
