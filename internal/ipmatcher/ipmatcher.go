// Package ipmatcher provides high-performance IP address matching using
// the bart radix tree library. It supports both IPv4 and IPv6 addresses,
// CIDR notation, and provides thread-safe operations for concurrent access.
//
// Key features:
// - High-performance IP matching using bart radix trees
// - Support for both IPv4 and IPv6 addresses
// - CIDR notation support for network ranges
// - Thread-safe operations with RWMutex
// - Separate whitelist and blocklist trees
// - Efficient memory usage and fast lookup times
package ipmatcher

import (
	"fmt"
	"net/netip"
	"strings"
	"sync"

	"github.com/gaissmai/bart"
)

// IPMatcher provides high-performance IP address matching using bart radix trees.
// It maintains separate trees for whitelist and blocklist entries, allowing
// efficient lookup and prioritizing whitelist checks over blocklist checks.
//
// The IPMatcher is thread-safe and can be used concurrently from multiple
// goroutines. It uses RWMutex to allow concurrent reads while ensuring
// exclusive access during writes.
type IPMatcher struct {
	whitelist *bart.Table[bool] // Radix tree for whitelisted IPs/CIDR blocks
	blocklist *bart.Table[bool] // Radix tree for blocked IPs/CIDR blocks
	mu        sync.RWMutex      // Mutex for thread-safe operations

	// Debug tracking (only populated when debugTrackingEnabled is true)
	debugTrackingEnabled bool     // Flag to enable/disable entry tracking
	whitelistEntries     []string // Original whitelist entries for debugging
	blocklistEntries     []string // Original blocklist entries for debugging
}

// NewIPMatcher creates a new IPMatcher instance with empty whitelist
// and blocklist trees. The trees are initialized but contain no entries.
//
// Returns:
//
//	*IPMatcher - A new IPMatcher instance ready for use
//
// Example:
//
//	matcher := NewIPMatcher()
//	err := matcher.LoadWhitelist(whitelistEntries)
//	err := matcher.LoadBlocklist(blocklistEntries)
func NewIPMatcher() *IPMatcher {
	return &IPMatcher{
		whitelist: &bart.Table[bool]{},
		blocklist: &bart.Table[bool]{},
	}
}

// LoadWhitelist loads IP addresses and CIDR blocks into the whitelist.
// This replaces any existing whitelist entries with the new ones.
//
// The function:
// - Acquires a write lock for thread safety
// - Clears the existing whitelist
// - Parses and adds each entry to the whitelist tree
// - Supports both single IP addresses and CIDR notation
// - Skips empty lines and comments
//
// Parameters:
//
//	entries - List of IP addresses or CIDR blocks to add to whitelist
//
// Returns:
//
//	error - Any error that occurred during parsing or insertion
//
// Example:
//
//	err := matcher.LoadWhitelist([]string{"198.51.100.1", "2001:db8::/32", "203.0.113.0/24"})
func (m *IPMatcher) LoadWhitelist(entries []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing whitelist
	m.whitelist = &bart.Table[bool]{}

	// Store entries for debugging if tracking is enabled
	if m.debugTrackingEnabled {
		m.whitelistEntries = make([]string, len(entries))
		copy(m.whitelistEntries, entries)
	}

	for _, entry := range entries {
		if err := m.addIPToTree(m.whitelist, entry); err != nil {
			return err
		}
	}

	return nil
}

// LoadBlocklist loads IP addresses and CIDR blocks into the blocklist.
// This replaces any existing blocklist entries with the new ones.
//
// The function:
// - Acquires a write lock for thread safety
// - Clears the existing blocklist
// - Parses and adds each entry to the blocklist tree
// - Supports both single IP addresses and CIDR notation
// - Skips empty lines and comments
//
// Parameters:
//
//	entries - List of IP addresses or CIDR blocks to add to blocklist
//
// Returns:
//
//	error - Any error that occurred during parsing or insertion
//
// Example:
//
//	err := matcher.LoadBlocklist([]string{"198.51.100.1", "2001:db8::/32", "203.0.113.0/24"})
func (m *IPMatcher) LoadBlocklist(entries []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing blocklist
	m.blocklist = &bart.Table[bool]{}

	// Store entries for debugging if tracking is enabled
	if m.debugTrackingEnabled {
		m.blocklistEntries = make([]string, len(entries))
		copy(m.blocklistEntries, entries)
	}

	for _, entry := range entries {
		if err := m.addIPToTree(m.blocklist, entry); err != nil {
			return err
		}
	}

	return nil
}

// addIPToTree adds an IP address or CIDR block to a bart radix tree.
// This is an internal helper function used by LoadWhitelist and LoadBlocklist.
//
// The function:
// - Trims whitespace and skips empty lines/comments
// - Parses CIDR notation or single IP addresses
// - Converts single IPs to appropriate prefixes (/32 for IPv4, /128 for IPv6)
// - Inserts the prefix into the specified tree
//
// Parameters:
//
//	tree - The bart radix tree to insert into
//	entry - The IP address or CIDR block to add
//
// Returns:
//
//	error - Any error that occurred during parsing
//
// Note: This function is internal and should not be called directly.
// Use LoadWhitelist or LoadBlocklist instead.
func (m *IPMatcher) addIPToTree(tree *bart.Table[bool], entry string) error {
	entry = strings.TrimSpace(entry)
	if entry == "" || strings.HasPrefix(entry, "#") {
		return nil
	}

	if strings.Contains(entry, "/") {
		// CIDR notation
		pfx, err := netip.ParsePrefix(entry)
		if err != nil {
			return fmt.Errorf("invalid CIDR: %s", entry)
		}
		tree.Insert(pfx, true)
	} else {
		// Single IP address
		addr, err := netip.ParseAddr(entry)
		if err != nil {
			return fmt.Errorf("invalid IP: %s", entry)
		}
		// Convert IP to /128 (IPv6) or /32 (IPv4) prefix
		pfx := netip.PrefixFrom(addr, addr.BitLen())
		tree.Insert(pfx, true)
	}
	return nil
}

// IsBlocked checks if an IP is blocked or whitelisted.
// This is the main lookup function that performs the actual IP matching.
//
// The function:
// - Acquires a read lock for thread-safe access
// - Parses the IP address
// - Checks whitelist first (whitelist takes precedence over blocklist)
// - Checks blocklist if not whitelisted
// - Returns whether the IP is blocked, the reason, and any error
//
// Parameters:
//
//	ipStr - The IP address to check (IPv4 or IPv6)
//
// Returns:
//
//	bool - true if the IP is blocked, false if allowed
//	string - Reason for the decision:
//	  - "whitelisted" if the IP is in the whitelist
//	  - "matched IP <ip>" if the IP is in the blocklist
//	  - empty string if the IP is neither blocked nor whitelisted
//	error - Any error that occurred during parsing
//
// Example:
//
//	blocked, reason, err := matcher.IsBlocked("198.51.100.1")
func (m *IPMatcher) IsBlocked(ipStr string) (bool, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return false, "", fmt.Errorf("invalid IP: %s", ipStr)
	}

	// Check whitelist first (whitelist takes precedence)
	if m.whitelist.Contains(addr) {
		return false, "whitelisted", nil
	}

	// Check blocklist
	if m.blocklist.Contains(addr) {
		return true, fmt.Sprintf("matched IP %s", ipStr), nil
	}

	return false, "", nil
}

// GetWhitelistSize returns the number of entries in the whitelist.
// This is useful for monitoring and debugging the size of the whitelist.
//
// Returns:
//
//	int - Number of entries in the whitelist
//
// This function is thread-safe and can be called concurrently.
func (m *IPMatcher) GetWhitelistSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for range m.whitelist.All() {
		count++
	}
	return count
}

// GetBlocklistSize returns the number of entries in the blocklist.
// This is useful for monitoring and debugging the size of the blocklist.
//
// Returns:
//
//	int - Number of entries in the blocklist
//
// This function is thread-safe and can be called concurrently.
func (m *IPMatcher) GetBlocklistSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for range m.blocklist.All() {
		count++
	}
	return count
}

// EnableDebugTracking enables tracking of loaded entries for debugging
// This should be called before loading any entries to ensure they are tracked
func (m *IPMatcher) EnableDebugTracking(enable bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.debugTrackingEnabled = enable
	if !enable {
		// Clear entries when disabling to free memory
		m.whitelistEntries = nil
		m.blocklistEntries = nil
	}
}

// GetWhitelistEntries returns the loaded whitelist entries (for debugging)
// Returns nil if debug tracking is not enabled
func (m *IPMatcher) GetWhitelistEntries() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.debugTrackingEnabled {
		return nil
	}
	// Return a copy to prevent external modification
	entries := make([]string, len(m.whitelistEntries))
	copy(entries, m.whitelistEntries)
	return entries
}

// GetBlocklistEntries returns the loaded blocklist entries (for debugging)
// Returns nil if debug tracking is not enabled
func (m *IPMatcher) GetBlocklistEntries() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.debugTrackingEnabled {
		return nil
	}
	// Return a copy to prevent external modification
	entries := make([]string, len(m.blocklistEntries))
	copy(entries, m.blocklistEntries)
	return entries
}

// ClearWhitelist clears all entries from the whitelist.
// This is useful when whitelist changes are detected and the
// entire whitelist needs to be reloaded.
//
// This function is thread-safe and acquires a write lock to ensure
// exclusive access during the clear operation.
func (m *IPMatcher) ClearWhitelist() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.whitelist = &bart.Table[bool]{}
}

// ClearBlocklist clears all entries from the blocklist.
// This is useful when blocklist changes are detected and the
// entire blocklist needs to be reloaded.
//
// This function is thread-safe and acquires a write lock to ensure
// exclusive access during the clear operation.
func (m *IPMatcher) ClearBlocklist() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blocklist = &bart.Table[bool]{}
}
