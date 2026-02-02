package cache

import (
	"testing"
	"time"
)

func TestCacheBasicOperations(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer func() { _ = c.Close() }()

	// Test Set and Get
	err = c.Set("test-key", "test-value")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, found := c.Get("test-key")
	if !found {
		t.Fatal("expected to find cached value")
	}
	if val != "test-value" {
		t.Errorf("got %v, want test-value", val)
	}

	// Test miss
	_, found = c.Get("nonexistent")
	if found {
		t.Error("expected cache miss for nonexistent key")
	}
}

func TestCacheStats(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer func() { _ = c.Close() }()

	// Generate some hits and misses
	_ = c.Set("key1", "value1")
	c.Get("key1") // hit
	c.Get("key1") // hit
	c.Get("key2") // miss

	stats := c.Stats()

	if stats["hits"].(int64) != 2 {
		t.Errorf("expected 2 hits, got %v", stats["hits"])
	}
	if stats["misses"].(int64) != 1 {
		t.Errorf("expected 1 miss, got %v", stats["misses"])
	}
}

func TestInvalidateMoods(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer func() { _ = c.Close() }()

	// Set some values
	_ = c.Set(KeyMoodsList, []string{"focus", "calm"})
	_ = c.Set(PlaylistKey("focus"), "focus-playlist")
	_ = c.Set(PlaylistKey("calm"), "calm-playlist")
	_ = c.Set("other-key", "other-value")

	// Verify they exist
	if _, found := c.Get(KeyMoodsList); !found {
		t.Fatal("moods list should exist before invalidation")
	}

	// Invalidate
	c.InvalidateMoods()

	// Verify mood-related keys are gone
	if _, found := c.Get(KeyMoodsList); found {
		t.Error("moods list should be invalidated")
	}
	if _, found := c.Get(PlaylistKey("focus")); found {
		t.Error("focus playlist should be invalidated")
	}
	if _, found := c.Get(PlaylistKey("calm")); found {
		t.Error("calm playlist should be invalidated")
	}

	// Other keys should still exist
	if _, found := c.Get("other-key"); !found {
		t.Error("other-key should NOT be invalidated")
	}
}

func TestCacheExpiry(t *testing.T) {
	c, err := New()
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}
	defer func() { _ = c.Close() }()

	// Manually insert an already-expired entry
	c.mu.Lock()
	c.items["expired"] = entry{value: "gone", expiresAt: time.Now().Add(-time.Second)}
	c.mu.Unlock()

	// Should not be found (expired on read)
	if _, found := c.Get("expired"); found {
		t.Error("expected expired value to not be returned")
	}
}
