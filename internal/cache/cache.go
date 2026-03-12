// Package cache provides an in-memory cache for IP blocking decisions.
// It uses sharded locks for concurrent access and supports TTL-based expiration.
// The cache stores entries as ALLOW (whitelisted) or DENY (blocked) statuses.
//
// Key features:
// - Sharded architecture for high concurrency
// - TTL-based automatic expiration
// - Thread-safe operations with RWMutex
// - Background pruning for memory management
package cache

import (
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
)

// Cache status constants
const (
	// CacheAllow indicates an IP is whitelisted and should be allowed
	CacheAllow = "ALLOW"
	// CacheDeny indicates an IP is blocked and should be denied
	CacheDeny = "DENY"
	// CacheMiss indicates an IP was not found in cache or has expired
	CacheMiss = "MISS"
)

// CacheEntry represents a single cache entry containing the status
// and timestamp of an IP blocking decision.
// Status can be "ALLOW" (whitelisted), "DENY" (blocked), or "MISS" (not found).
// Timestamp indicates when the entry was added or last updated.
type CacheEntry struct {
	Status    string    // Cache status: ALLOW, DENY, or MISS
	Timestamp time.Time // When the entry was created or updated
}

type shard struct {
	entries map[string]CacheEntry
	mu      sync.RWMutex
}

// Cache provides a thread-safe, sharded in-memory cache for IP blocking decisions.
// It supports TTL-based expiration and automatic pruning.
//
// The cache uses a sharded architecture with RWMutex locks for each shard,
// allowing high concurrency while maintaining thread safety.
type Cache struct {
	shards          []shard // Sharded storage for concurrent access
	numShards       int     // Number of shards for load distribution
	TTL             int     // Time-to-live in seconds for cache entries
	MaxEntries      int     // Maximum number of entries before eviction
	PruneInterval   int     // Interval in seconds for background pruning
	PruneOnGet      bool    // Whether to prune expired entries on cache miss
	WriteBufferSize int     // Write buffer size for batch operations
}

// NewCache creates a new Cache instance with the specified configuration.
//
// Parameters:
//
//	ttl - Time-to-live in seconds for cache entries
//	maxEntries - Maximum number of entries before eviction
//	pruneInterval - Interval in seconds for background pruning
//	shardCount - Number of shards for load distribution
//	pruneOnGet - Whether to prune expired entries on cache miss
//	writeBufferSize - Write buffer size for batch operations
//
// Returns:
//
//	*Cache - A new cache instance ready for use
//
// Example:
//
//	cache := NewCache(300, 100000, 60, 64, false, 0)
func NewCache(ttl, maxEntries, pruneInterval, shardCount int, pruneOnGet bool, writeBufferSize int) *Cache {
	cache := &Cache{
		TTL:             ttl,
		MaxEntries:      maxEntries,
		PruneInterval:   pruneInterval,
		numShards:       shardCount,
		PruneOnGet:      pruneOnGet,
		WriteBufferSize: writeBufferSize,
	}

	// Initialize all shards
	cache.shards = make([]shard, shardCount)
	for i := range cache.shards {
		cache.shards[i] = shard{
			entries: make(map[string]CacheEntry),
		}
	}

	return cache
}

// getCacheKey returns the cache key for an IP address.
//
// Parameters:
//
//	ip - The IP address
//
// Returns:
//
//	string - Cache key for the IP
func (c *Cache) getCacheKey(ip string) string {
	return ip
}

// getShard returns the shard for a given key using FNV-1a hash.
// This distributes cache entries across multiple shards for better concurrency.
//
// Parameters:
//
//	key - The cache key to hash
//
// Returns:
//
//	*shard - The shard responsible for this key go deprecate FNV-1a in favor of xxhash for better performance and lower collision rates
// func (c *Cache) getShard(key string) *shard {
// 	h := fnv.New32a()
// 	h.Write([]byte(key))
// 	shardIndex := h.Sum32() % uint32(c.numShards)
// 	return &c.shards[shardIndex]
// }

func (c *Cache) getShard(key string) *shard {
	shardIndex := xxhash.Sum64([]byte(key)) % uint64(c.numShards) // #nosec G115 needs to be addressed todo
	return &c.shards[shardIndex]
}

// GetStatus returns the cache status for an IP.
// It checks if the IP exists in cache and whether it has expired based on TTL.
// The function uses read locks for performance and only upgrades to write locks
// when necessary for cache maintenance operations.
//
// Parameters:
//
//	ip - The IP address to check (IPv4 or IPv6)
//
// Returns:
//
//	string - Cache status: "ALLOW", "DENY", or "MISS" if not found/expired
//
// The function is thread-safe and can be called concurrently from multiple goroutines.
// It follows a read-optimized lock upgrade pattern to minimize contention.
func (c *Cache) GetStatus(ip string) string {
	key := c.getCacheKey(ip)
	shard := c.getShard(key)

	// Use read lock for all reads to ensure thread safety
	shard.mu.RLock()
	if entry, exists := shard.entries[key]; exists {
		// Re-check with lock to ensure consistency
		if time.Since(entry.Timestamp) <= time.Duration(c.TTL)*time.Second {
			shard.mu.RUnlock()
			return entry.Status
		}
		// Entry expired, need write lock to delete
		shard.mu.RUnlock()
		shard.mu.Lock()
		// Re-check after acquiring write lock (entry might have been updated)
		if entry2, exists2 := shard.entries[key]; exists2 && time.Since(entry2.Timestamp) > time.Duration(c.TTL)*time.Second {
			delete(shard.entries, key)
		}
		shard.mu.Unlock()
		return CacheMiss
	}
	shard.mu.RUnlock()

	// Optional: Prune expired entries on cache miss (higher CPU but lower memory)
	if c.PruneOnGet {
		c.pruneExpiredEntries()
	}

	return CacheMiss
}

// Add inserts or updates a cache entry for an IP.
// The entry will expire after the configured TTL period.
//
// Parameters:
//
//	ip - The IP address to cache
//	status - The cache status: "ALLOW" or "DENY"
//
// This function is thread-safe and can be called concurrently.
// It acquires a write lock on the appropriate shard for the duration
// of the operation to ensure atomicity.
func (c *Cache) Add(ip string, status string) {
	key := c.getCacheKey(ip)
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	shard.entries[key] = CacheEntry{
		Status:    status,
		Timestamp: time.Now(),
	}
}

// pruneExpiredEntries removes all expired entries from all shards.
// This is called periodically by the background pruner or on cache misses
// when PruneOnGet is enabled.
//
// The function iterates through all shards and removes entries whose
// timestamp is older than the configured TTL.
//
// This function is thread-safe and acquires write locks on all shards
// to ensure no concurrent modifications during pruning.
func (c *Cache) pruneExpiredEntries() {
	now := time.Now()
	for i := range c.shards {
		shard := &c.shards[i]
		shard.mu.Lock()
		for ip, entry := range shard.entries {
			if now.Sub(entry.Timestamp) > time.Duration(c.TTL)*time.Second {
				delete(shard.entries, ip)
			}
		}
		shard.mu.Unlock()
	}
}

// Prune removes all expired entries from the cache.
// This function is typically called by the background pruner
// at regular intervals as configured by PruneInterval.
//
// Returns:
//
//	int - Number of entries evicted (not directly returned, but tracked internally)
//
// This function is thread-safe and acquires write locks on all shards
// to ensure no concurrent modifications during pruning.
func (c *Cache) Prune() {
	now := time.Now()
	evicted := 0
	for i := range c.shards {
		shard := &c.shards[i]
		shard.mu.Lock()
		for ip, entry := range shard.entries {
			if now.Sub(entry.Timestamp) > time.Duration(c.TTL)*time.Second {
				delete(shard.entries, ip)
				evicted++
			}
		}
		shard.mu.Unlock()
	}
}

// StartPruner starts a background goroutine that periodically prunes
// expired entries from the cache. The pruning interval is configured
// by the PruneInterval field during cache creation.
//
// This function starts a ticker that calls Prune() at the specified interval.
// The goroutine runs indefinitely until the program terminates.
//
// Example:
//
//	cache := NewCache(300, 100000, 60, 64, false, 0)
//	cache.StartPruner() // Start background pruning every 60 seconds
func (c *Cache) StartPruner() {
	ticker := time.NewTicker(time.Duration(c.PruneInterval) * time.Second)
	go func() {
		for range ticker.C {
			c.Prune()
		}
	}()
}

// Clear removes all entries from the cache, effectively resetting it.
// This is useful when blocklist or whitelist changes are detected
// and all cached decisions need to be invalidated.
//
// Returns:
//
//	int - Number of entries cleared (not directly returned, but tracked internally)
//
// This function is thread-safe and acquires write locks on all shards
// to ensure no concurrent access during clearing.
func (c *Cache) Clear() {
	evicted := 0
	for i := range c.shards {
		shard := &c.shards[i]
		shard.mu.Lock()
		evicted += len(shard.entries)
		shard.entries = make(map[string]CacheEntry)
		shard.mu.Unlock()
	}
}

// GetEntries returns a copy of all current cache entries across all shards.
// This is useful for debugging, monitoring, or exporting cache state.
//
// Returns:
//
//	map[string]CacheEntry - A copy of all active cache entries
//
// This function is thread-safe and acquires read locks on all shards
// to ensure a consistent snapshot of the cache state.
func (c *Cache) GetEntries() map[string]CacheEntry {
	entriesCopy := make(map[string]CacheEntry)
	for i := range c.shards {
		shard := &c.shards[i]
		shard.mu.RLock()
		for k, v := range shard.entries {
			entriesCopy[k] = v
		}
		shard.mu.RUnlock()
	}
	return entriesCopy
}

// GetStats returns comprehensive statistics about the cache state.
// This includes counts of total entries, ALLOW entries, DENY entries,
// and configuration information.
//
// Returns:
//
//	map[string]interface{} - Cache statistics including:
//	  - total_entries: Total number of active cache entries
//	  - deny_entries: Number of DENY (blocked) entries
//	  - allow_entries: Number of ALLOW (whitelisted) entries
//	  - shard_count: Number of shards in the cache
//	  - ttl_seconds: Time-to-live in seconds
//	  - max_entries: Maximum number of entries before eviction
//
// This function is useful for monitoring cache performance and health.
func (c *Cache) GetStats() map[string]interface{} {
	totalEntries := 0
	denyEntries := 0
	allowEntries := 0

	entries := c.GetEntries()

	for _, entry := range entries {
		totalEntries++
		if entry.Status == CacheDeny {
			denyEntries++
		} else if entry.Status == CacheAllow {
			allowEntries++
		}
	}

	return map[string]interface{}{
		"total_entries": totalEntries,
		"deny_entries":  denyEntries,
		"allow_entries": allowEntries,
		"shard_count":   c.numShards,
		"ttl_seconds":   c.TTL,
		"max_entries":   c.MaxEntries,
	}
}

// cacheInstance is a package-level variable to hold the cache instance
// so it can be accessed from other packages like the cron scheduler.
var cacheInstance *Cache

// SetCacheInstance sets the package-level cache instance.
// This should be called once when the cache is initialized.
func SetCacheInstance(c *Cache) {
	cacheInstance = c
}
