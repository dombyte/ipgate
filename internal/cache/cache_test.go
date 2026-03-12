package cache

import (
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	// Test with default parameters
	cache := NewCache(300, 100000, 60, 64, false, 0)
	if cache == nil {
		t.Fatal("Expected non-nil cache")
	}
	if cache.TTL != 300 {
		t.Errorf("Expected TTL 300, got %d", cache.TTL)
	}
	if cache.shards == nil {
		t.Fatal("Expected non-nil shards")
	}
	if len(cache.shards) != 64 {
		t.Errorf("Expected 64 shards, got %d", len(cache.shards))
	}
}

func TestCache_GetStatus(t *testing.T) {
	cache := NewCache(300, 100000, 60, 64, false, 0)

	// Test cache miss for non-existent IP
	status := cache.GetStatus("198.51.100.1")
	if status != CacheMiss {
		t.Errorf("Expected CacheMiss, got %s", status)
	}

	// Test adding to cache
	cache.Add("198.51.100.1", CacheDeny)
	status = cache.GetStatus("198.51.100.1")
	if status != CacheDeny {
		t.Errorf("Expected CacheDeny, got %s", status)
	}

	// Test adding allow status
	cache.Add("10.0.0.1", CacheAllow)
	status = cache.GetStatus("10.0.0.1")
	if status != CacheAllow {
		t.Errorf("Expected CacheAllow, got %s", status)
	}
}

func TestCache_Add(t *testing.T) {
	cache := NewCache(300, 100000, 60, 64, false, 0)

	// Test adding deny entry
	cache.Add("198.51.100.1", CacheDeny)
	status := cache.GetStatus("198.51.100.1")
	if status != CacheDeny {
		t.Errorf("Expected CacheDeny, got %s", status)
	}

	// Test adding allow entry
	cache.Add("10.0.0.1", CacheAllow)
	status = cache.GetStatus("10.0.0.1")
	if status != CacheAllow {
		t.Errorf("Expected CacheAllow, got %s", status)
	}
}

func TestCache_Prune(t *testing.T) {
	cache := NewCache(1, 100000, 60, 64, false, 0) // 1 second TTL for testing

	// Add entries
	cache.Add("198.51.100.1", CacheDeny)
	cache.Add("10.0.0.1", CacheAllow)

	// Verify entries exist
	status := cache.GetStatus("198.51.100.1")
	if status != CacheDeny {
		t.Errorf("Expected CacheDeny before prune, got %s", status)
	}

	// Wait for entries to expire
	time.Sleep(2 * time.Second)

	// Prune expired entries
	cache.Prune()

	// Verify entries are removed
	status = cache.GetStatus("198.51.100.1")
	if status != CacheMiss {
		t.Errorf("Expected CacheMiss after prune, got %s", status)
	}

	status = cache.GetStatus("10.0.0.1")
	if status != CacheMiss {
		t.Errorf("Expected CacheMiss after prune, got %s", status)
	}
}

func TestCache_Clear(t *testing.T) {
	cache := NewCache(300, 100000, 60, 64, false, 0)

	// Add entries
	cache.Add("198.51.100.1", CacheDeny)
	cache.Add("10.0.0.1", CacheAllow)

	// Verify entries exist
	status := cache.GetStatus("198.51.100.1")
	if status != CacheDeny {
		t.Errorf("Expected CacheDeny before clear, got %s", status)
	}

	// Clear cache
	cache.Clear()

	// Verify entries are removed
	status = cache.GetStatus("198.51.100.1")
	if status != CacheMiss {
		t.Errorf("Expected CacheMiss after clear, got %s", status)
	}

	status = cache.GetStatus("10.0.0.1")
	if status != CacheMiss {
		t.Errorf("Expected CacheMiss after clear, got %s", status)
	}
}

func TestCache_GetEntries(t *testing.T) {
	cache := NewCache(300, 100000, 60, 64, false, 0)

	// Test with empty cache
	entries := cache.GetEntries()
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries in empty cache, got %d", len(entries))
	}

	// Add entries
	cache.Add("198.51.100.1", CacheDeny)
	cache.Add("10.0.0.1", CacheAllow)

	// Get entries
	entries = cache.GetEntries()
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}

	// Verify entry details
	foundDeny := false
	foundAllow := false
	for ip, entry := range entries {
		if ip == "198.51.100.1" && entry.Status == CacheDeny {
			foundDeny = true
		}
		if ip == "10.0.0.1" && entry.Status == CacheAllow {
			foundAllow = true
		}
	}

	if !foundDeny {
		t.Error("Expected to find deny entry")
	}
	if !foundAllow {
		t.Error("Expected to find allow entry")
	}
}

func TestCache_GetStats(t *testing.T) {
	cache := NewCache(300, 100000, 60, 64, false, 0)

	// Test with empty cache
	stats := cache.GetStats()
	totalEntries, ok := stats["total_entries"].(int)
	if !ok || totalEntries != 0 {
		t.Errorf("Expected 0 total entries, got %d", totalEntries)
	}

	// Add entries
	cache.Add("198.51.100.1", CacheDeny)
	cache.Add("10.0.0.1", CacheAllow)

	// Get stats
	stats = cache.GetStats()
	totalEntries, ok = stats["total_entries"].(int)
	if !ok || totalEntries != 2 {
		t.Errorf("Expected 2 total entries, got %d", totalEntries)
	}
}

func TestCache_ThreadSafety(t *testing.T) {
	cache := NewCache(300, 100000, 60, 64, false, 0)

	// Add initial entries
	cache.Add("198.51.100.1", CacheDeny)
	cache.Add("10.0.0.1", CacheAllow)

	// Test concurrent reads
	done := make(chan bool, 2)

	// Goroutine 1: Read operations
	go func() {
		for i := 0; i < 1000; i++ {
			_ = cache.GetStatus("198.51.100.1")
			_ = cache.GetStatus("10.0.0.1")
			_ = cache.GetStatus("203.0.113.1")
		}
		done <- true
	}()

	// Goroutine 2: Read operations
	go func() {
		for i := 0; i < 1000; i++ {
			_ = cache.GetStatus("172.16.0.1")
			_ = cache.GetStatus("192.168.1.1")
			_ = cache.GetStatus("10.0.0.1")
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	for i := 0; i < 2; i++ {
		<-done
	}

	// Verify data integrity after concurrent reads
	status := cache.GetStatus("198.51.100.1")
	if status != CacheDeny {
		t.Errorf("Expected CacheDeny after concurrent reads, got %s", status)
	}

	status = cache.GetStatus("10.0.0.1")
	if status != CacheAllow {
		t.Errorf("Expected CacheAllow after concurrent reads, got %s", status)
	}
}

func TestCache_Sharding(t *testing.T) {
	cache := NewCache(300, 100000, 60, 64, false, 0)

	// Test that different IPs go to different shards
	// This is implicit in the implementation, but we can verify
	// that the same IP always goes to the same shard
	for i := 0; i < 10; i++ {
		cache.Add("198.51.100.1", CacheDeny)
		status := cache.GetStatus("198.51.100.1")
		if status != CacheDeny {
			t.Errorf("Expected CacheDeny on iteration %d, got %s", i, status)
		}
	}
}

func TestCache_InvalidIP(t *testing.T) {
	cache := NewCache(300, 100000, 60, 64, false, 0)

	// Test with invalid IP (cache accepts any string as key)
	status := cache.GetStatus("invalid-ip")
	if status != CacheMiss {
		t.Errorf("Expected CacheMiss for non-existent IP, got %s", status)
	}

	// Test adding invalid IP (cache accepts any string as key)
	cache.Add("invalid-ip", CacheDeny)
	status = cache.GetStatus("invalid-ip")
	if status != CacheDeny {
		t.Errorf("Expected CacheDeny for added IP, got %s", status)
	}
}

func TestCache_EntryExpiration(t *testing.T) {
	cache := NewCache(1, 100000, 60, 64, false, 0) // 1 second TTL

	// Add entry
	cache.Add("198.51.100.1", CacheDeny)
	status := cache.GetStatus("198.51.100.1")
	if status != CacheDeny {
		t.Errorf("Expected CacheDeny immediately after add, got %s", status)
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Entry should be expired
	status = cache.GetStatus("198.51.100.1")
	if status != CacheMiss {
		t.Errorf("Expected CacheMiss after expiration, got %s", status)
	}
}

func TestCache_UpdateExistingEntry(t *testing.T) {
	cache := NewCache(300, 100000, 60, 64, false, 0)

	// Add deny entry
	cache.Add("198.51.100.1", CacheDeny)
	status := cache.GetStatus("198.51.100.1")
	if status != CacheDeny {
		t.Errorf("Expected CacheDeny, got %s", status)
	}

	// Update to allow
	cache.Add("198.51.100.1", CacheAllow)
	status = cache.GetStatus("198.51.100.1")
	if status != CacheAllow {
		t.Errorf("Expected CacheAllow after update, got %s", status)
	}
}
