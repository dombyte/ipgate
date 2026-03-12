package ipmatcher

import (
	"testing"
)

// TestEnableDebugTracking tests the EnableDebugTracking method
func TestEnableDebugTracking(t *testing.T) {
	m := NewIPMatcher()

	// Test enabling debug tracking
	m.EnableDebugTracking(true)

	// Load some entries
	entries := []string{"198.51.100.1", "2001:db8::/32"}
	if err := m.LoadWhitelist(entries); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	if err := m.LoadBlocklist(entries); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	// Verify entries are tracked
	whitelistEntries := m.GetWhitelistEntries()
	if whitelistEntries == nil {
		t.Fatal("Whitelist entries should not be nil when debug tracking is enabled")
	}

	if len(whitelistEntries) != 2 {
		t.Errorf("Expected 2 whitelist entries, got %d", len(whitelistEntries))
	}

	blocklistEntries := m.GetBlocklistEntries()
	if blocklistEntries == nil {
		t.Fatal("Blocklist entries should not be nil when debug tracking is enabled")
	}

	if len(blocklistEntries) != 2 {
		t.Errorf("Expected 2 blocklist entries, got %d", len(blocklistEntries))
	}

	// Test disabling debug tracking
	m.EnableDebugTracking(false)

	whitelistEntries = m.GetWhitelistEntries()
	if whitelistEntries != nil {
		t.Error("Whitelist entries should be nil when debug tracking is disabled")
	}

	blocklistEntries = m.GetBlocklistEntries()
	if blocklistEntries != nil {
		t.Error("Blocklist entries should be nil when debug tracking is disabled")
	}
}

// TestGetWhitelistEntries tests the GetWhitelistEntries method
func TestGetWhitelistEntries(t *testing.T) {
	m := NewIPMatcher()

	// Test without debug tracking
	entries := []string{"198.51.100.1", "203.0.113.0/24"}
	if err := m.LoadWhitelist(entries); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	whitelistEntries := m.GetWhitelistEntries()
	if whitelistEntries != nil {
		t.Error("Whitelist entries should be nil when debug tracking is not enabled")
	}

	// Test with debug tracking
	m.EnableDebugTracking(true)
	m.ClearWhitelist() // Clear previous entries
	if err := m.LoadWhitelist(entries); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	whitelistEntries = m.GetWhitelistEntries()
	if whitelistEntries == nil {
		t.Fatal("Whitelist entries should not be nil when debug tracking is enabled")
	}

	if len(whitelistEntries) != 2 {
		t.Errorf("Expected 2 whitelist entries, got %d", len(whitelistEntries))
	}

	// Verify entries are correct
	if whitelistEntries[0] != "198.51.100.1" {
		t.Errorf("Expected first entry to be '198.51.100.1', got '%s'", whitelistEntries[0])
	}

	if whitelistEntries[1] != "203.0.113.0/24" {
		t.Errorf("Expected second entry to be '203.0.113.0/24', got '%s'", whitelistEntries[1])
	}
}

// TestGetBlocklistEntries tests the GetBlocklistEntries method
func TestGetBlocklistEntries(t *testing.T) {
	m := NewIPMatcher()

	// Test without debug tracking
	entries := []string{"198.51.100.2", "2001:db8::/32"}
	if err := m.LoadBlocklist(entries); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	blocklistEntries := m.GetBlocklistEntries()
	if blocklistEntries != nil {
		t.Error("Blocklist entries should be nil when debug tracking is not enabled")
	}

	// Test with debug tracking
	m.EnableDebugTracking(true)
	m.ClearBlocklist() // Clear previous entries
	if err := m.LoadBlocklist(entries); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	blocklistEntries = m.GetBlocklistEntries()
	if blocklistEntries == nil {
		t.Fatal("Blocklist entries should not be nil when debug tracking is enabled")
	}

	if len(blocklistEntries) != 2 {
		t.Errorf("Expected 2 blocklist entries, got %d", len(blocklistEntries))
	}

	// Verify entries are correct
	if blocklistEntries[0] != "198.51.100.2" {
		t.Errorf("Expected first entry to be '198.51.100.2', got '%s'", blocklistEntries[0])
	}

	if blocklistEntries[1] != "2001:db8::/32" {
		t.Errorf("Expected second entry to be '2001:db8::/32', got '%s'", blocklistEntries[1])
	}
}

// TestClearWhitelist tests the ClearWhitelist method
func TestClearWhitelist(t *testing.T) {
	m := NewIPMatcher()

	// Load whitelist
	entries := []string{"198.51.100.1", "203.0.113.0/24"}
	if err := m.LoadWhitelist(entries); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	// Verify whitelist is not empty
	size := m.GetWhitelistSize()
	if size != 2 {
		t.Errorf("Expected whitelist size 2, got %d", size)
	}

	// Clear whitelist
	m.ClearWhitelist()

	// Verify whitelist is empty
	size = m.GetWhitelistSize()
	if size != 0 {
		t.Errorf("Expected whitelist size 0 after clear, got %d", size)
	}

	// Test IsBlocked after clearing
	blocked, reason, err := m.IsBlocked("198.51.100.1")
	if err != nil {
		t.Fatalf("IsBlocked failed: %v", err)
	}

	if blocked {
		t.Error("Expected IP to not be blocked after clearing whitelist")
	}

	if reason != "" {
		t.Errorf("Expected empty reason, got '%s'", reason)
	}
}

// TestClearBlocklist tests the ClearBlocklist method
func TestClearBlocklist(t *testing.T) {
	m := NewIPMatcher()

	// Load blocklist
	entries := []string{"198.51.100.2", "2001:db8::/32"}
	if err := m.LoadBlocklist(entries); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	// Verify blocklist is not empty
	size := m.GetBlocklistSize()
	if size != 2 {
		t.Errorf("Expected blocklist size 2, got %d", size)
	}

	// Clear blocklist
	m.ClearBlocklist()

	// Verify blocklist is empty
	size = m.GetBlocklistSize()
	if size != 0 {
		t.Errorf("Expected blocklist size 0 after clear, got %d", size)
	}

	// Test IsBlocked after clearing
	blocked, reason, err := m.IsBlocked("198.51.100.2")
	if err != nil {
		t.Fatalf("IsBlocked failed: %v", err)
	}

	if blocked {
		t.Error("Expected IP to not be blocked after clearing blocklist")
	}

	if reason != "" {
		t.Errorf("Expected empty reason, got '%s'", reason)
	}
}

// TestLoadWhitelistWithEmptyEntries tests loading whitelist with empty entries
func TestLoadWhitelistWithEmptyEntries(t *testing.T) {
	m := NewIPMatcher()

	entries := []string{"", "   ", "# comment", "198.51.100.1"}
	if err := m.LoadWhitelist(entries); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	size := m.GetWhitelistSize()
	if size != 1 {
		t.Errorf("Expected whitelist size 1 (only valid IP), got %d", size)
	}

	// Verify the valid IP is whitelisted
	blocked, reason, err := m.IsBlocked("198.51.100.1")
	if err != nil {
		t.Fatalf("IsBlocked failed: %v", err)
	}

	if blocked {
		t.Error("Expected IP to be whitelisted")
	}

	if reason != "whitelisted" {
		t.Errorf("Expected reason 'whitelisted', got '%s'", reason)
	}
}

// TestLoadBlocklistWithEmptyEntries tests loading blocklist with empty entries
func TestLoadBlocklistWithEmptyEntries(t *testing.T) {
	m := NewIPMatcher()

	entries := []string{"", "   ", "# comment", "198.51.100.2"}
	if err := m.LoadBlocklist(entries); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	size := m.GetBlocklistSize()
	if size != 1 {
		t.Errorf("Expected blocklist size 1 (only valid IP), got %d", size)
	}

	// Verify the valid IP is blocked
	blocked, reason, err := m.IsBlocked("198.51.100.2")
	if err != nil {
		t.Fatalf("IsBlocked failed: %v", err)
	}

	if !blocked {
		t.Error("Expected IP to be blocked")
	}

	if reason != "matched IP 198.51.100.2" {
		t.Errorf("Expected reason 'matched IP 198.51.100.2', got '%s'", reason)
	}
}

// TestLoadWhitelistWithInvalidIPs tests loading whitelist with invalid IPs
func TestLoadWhitelistWithInvalidIPs(t *testing.T) {
	m := NewIPMatcher()

	entries := []string{"198.51.100.1", "invalid-ip", "2001:db8::/129"}
	err := m.LoadWhitelist(entries)

	if err == nil {
		t.Error("Expected error when loading invalid CIDR")
	}

	// Note: The function continues loading after error, so we expect 1 valid entry
	size := m.GetWhitelistSize()
	if size != 1 {
		t.Errorf("Expected whitelist size 1 (only valid IP), got %d", size)
	}
}

// TestLoadBlocklistWithInvalidIPs tests loading blocklist with invalid IPs
func TestLoadBlocklistWithInvalidIPs(t *testing.T) {
	m := NewIPMatcher()

	entries := []string{"198.51.100.2", "invalid-ip", "2001:db8::/129"}
	err := m.LoadBlocklist(entries)

	if err == nil {
		t.Error("Expected error when loading invalid CIDR")
	}

	// Note: The function continues loading after error, so we expect 1 valid entry
	size := m.GetBlocklistSize()
	if size != 1 {
		t.Errorf("Expected blocklist size 1 (only valid IP), got %d", size)
	}
}

// TestIsBlockedWithInvalidIP tests IsBlocked with invalid IP
func TestIsBlockedWithInvalidIP(t *testing.T) {
	m := NewIPMatcher()

	blocked, reason, err := m.IsBlocked("invalid-ip")

	if err == nil {
		t.Error("Expected error when parsing invalid IP")
	}

	if blocked {
		t.Error("Expected IP to not be blocked when parsing fails")
	}

	if reason != "" {
		t.Errorf("Expected empty reason for invalid IP, got '%s'", reason)
	}
}

// TestWhitelistTakesPrecedence tests that whitelist takes precedence over blocklist
func TestWhitelistTakesPrecedence(t *testing.T) {
	m := NewIPMatcher()

	// Load both whitelist and blocklist with same IP
	whitelistEntries := []string{"198.51.100.1"}
	blocklistEntries := []string{"198.51.100.1", "198.51.100.2"}

	if err := m.LoadWhitelist(whitelistEntries); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	if err := m.LoadBlocklist(blocklistEntries); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	// Verify whitelisted IP is not blocked
	blocked, reason, err := m.IsBlocked("198.51.100.1")
	if err != nil {
		t.Fatalf("IsBlocked failed: %v", err)
	}

	if blocked {
		t.Error("Expected IP to not be blocked (whitelist takes precedence)")
	}

	if reason != "whitelisted" {
		t.Errorf("Expected reason 'whitelisted', got '%s'", reason)
	}

	// Verify blocked IP is still blocked
	blocked, reason, err = m.IsBlocked("198.51.100.2")
	if err != nil {
		t.Fatalf("IsBlocked failed: %v", err)
	}

	if !blocked {
		t.Error("Expected IP to be blocked")
	}

	if reason != "matched IP 198.51.100.2" {
		t.Errorf("Expected reason 'matched IP 198.51.100.2', got '%s'", reason)
	}
}

// TestGetWhitelistSize tests the GetWhitelistSize method
func TestGetWhitelistSize(t *testing.T) {
	m := NewIPMatcher()

	// Test empty whitelist
	size := m.GetWhitelistSize()
	if size != 0 {
		t.Errorf("Expected whitelist size 0, got %d", size)
	}

	// Load whitelist
	entries := []string{"198.51.100.1", "203.0.113.0/24", "2001:db8::/32"}
	if err := m.LoadWhitelist(entries); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	size = m.GetWhitelistSize()
	if size != 3 {
		t.Errorf("Expected whitelist size 3, got %d", size)
	}
}

// TestGetBlocklistSize tests the GetBlocklistSize method
func TestGetBlocklistSize(t *testing.T) {
	m := NewIPMatcher()

	// Test empty blocklist
	size := m.GetBlocklistSize()
	if size != 0 {
		t.Errorf("Expected blocklist size 0, got %d", size)
	}

	// Load blocklist
	entries := []string{"198.51.100.2", "192.168.0.0/16", "2001:db8::/32"}
	if err := m.LoadBlocklist(entries); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	size = m.GetBlocklistSize()
	if size != 3 {
		t.Errorf("Expected blocklist size 3, got %d", size)
	}
}
