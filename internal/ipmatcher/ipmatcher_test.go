package ipmatcher

import (
	"testing"
)

func TestIPMatcher_Basic(t *testing.T) {
	matcher := NewIPMatcher()

	// Test with empty lists
	blocked, reason, err := matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to not be blocked with empty lists")
	}
	if reason != "" {
		t.Errorf("Expected empty reason, got: %s", reason)
	}
}

func TestIPMatcher_Whitelist(t *testing.T) {
	matcher := NewIPMatcher()

	// Load whitelist
	err := matcher.LoadWhitelist([]string{"198.51.100.1", "203.0.113.0/24"})
	if err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	// Test whitelisted IP
	blocked, reason, err := matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to be whitelisted")
	}
	if reason != "whitelisted" {
		t.Errorf("Expected reason 'whitelisted', got: %s", reason)
	}

	// Test whitelisted CIDR
	blocked, reason, err = matcher.IsBlocked("203.0.113.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to be whitelisted via CIDR")
	}
	if reason != "whitelisted" {
		t.Errorf("Expected reason 'whitelisted', got: %s", reason)
	}

	// Test non-whitelisted IP
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to not be blocked")
	}
}

func TestIPMatcher_Blocklist(t *testing.T) {
	matcher := NewIPMatcher()

	// Load blocklist
	err := matcher.LoadBlocklist([]string{"198.51.100.1", "203.0.113.0/24"})
	if err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	// Test blocked IP
	blocked, reason, err := matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !blocked {
		t.Error("Expected IP to be blocked")
	}
	if reason == "" {
		t.Error("Expected non-empty reason")
	}

	// Test blocked CIDR
	blocked, reason, err = matcher.IsBlocked("203.0.113.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !blocked {
		t.Error("Expected IP to be blocked via CIDR")
	}
	if reason == "" {
		t.Error("Expected non-empty reason")
	}

	// Test non-blocked IP
	blocked, reason, err = matcher.IsBlocked("10.0.0.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to not be blocked")
	}
}

func TestIPMatcher_WhitelistTakesPrecedence(t *testing.T) {
	matcher := NewIPMatcher()

	// Load both whitelist and blocklist
	err := matcher.LoadWhitelist([]string{"198.51.100.1"})
	if err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	err = matcher.LoadBlocklist([]string{"198.51.100.1"})
	if err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	// Test that whitelist takes precedence
	blocked, reason, err := matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to be whitelisted (whitelist should take precedence)")
	}
	if reason != "whitelisted" {
		t.Errorf("Expected reason 'whitelisted', got: %s", reason)
	}
}

func TestIPMatcher_InvalidIP(t *testing.T) {
	matcher := NewIPMatcher()

	_, _, err := matcher.IsBlocked("invalid-ip")
	if err == nil {
		t.Error("Expected error for invalid IP")
	}
}

func TestIPMatcher_IPv6(t *testing.T) {
	matcher := NewIPMatcher()

	// Load IPv6 entries
	err := matcher.LoadWhitelist([]string{"2001:db8::1"})
	if err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	err = matcher.LoadBlocklist([]string{"2001:db8::2", "2001:db8:1::/48"})
	if err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	// Test whitelisted IPv6
	blocked, _, err := matcher.IsBlocked("2001:db8::1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IPv6 to be whitelisted")
	}

	// Test blocked IPv6
	blocked, _, err = matcher.IsBlocked("2001:db8::2")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !blocked {
		t.Error("Expected IPv6 to be blocked")
	}

	// Test blocked via CIDR
	blocked, _, err = matcher.IsBlocked("2001:db8:1::1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !blocked {
		t.Error("Expected IPv6 to be blocked via CIDR")
	}
}

func TestIPMatcher_GetWhitelistSize(t *testing.T) {
	matcher := NewIPMatcher()

	// Test with empty whitelist
	size := matcher.GetWhitelistSize()
	if size != 0 {
		t.Errorf("Expected whitelist size 0, got %d", size)
	}

	// Load whitelist with single IP
	err := matcher.LoadWhitelist([]string{"198.51.100.1"})
	if err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	size = matcher.GetWhitelistSize()
	if size != 1 {
		t.Errorf("Expected whitelist size 1, got %d", size)
	}

	// Load whitelist with multiple IPs and CIDR
	err = matcher.LoadWhitelist([]string{"198.51.100.1", "203.0.113.0/24", "2001:db8::1", "2001:db8::/32"})
	if err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	size = matcher.GetWhitelistSize()
	if size != 4 {
		t.Errorf("Expected whitelist size 4, got %d", size)
	}

	// Test with comments and empty lines
	err = matcher.LoadWhitelist([]string{"# comment", "", "198.51.100.1", ""})
	if err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	size = matcher.GetWhitelistSize()
	if size != 1 {
		t.Errorf("Expected whitelist size 1 (comments and empty lines should be skipped), got %d", size)
	}
}

func TestIPMatcher_GetBlocklistSize(t *testing.T) {
	matcher := NewIPMatcher()

	// Test with empty blocklist
	size := matcher.GetBlocklistSize()
	if size != 0 {
		t.Errorf("Expected blocklist size 0, got %d", size)
	}

	// Load blocklist with single IP
	err := matcher.LoadBlocklist([]string{"198.51.100.1"})
	if err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	size = matcher.GetBlocklistSize()
	if size != 1 {
		t.Errorf("Expected blocklist size 1, got %d", size)
	}

	// Load blocklist with multiple IPs and CIDR
	err = matcher.LoadBlocklist([]string{"198.51.100.1", "203.0.113.0/24", "2001:db8::1", "2001:db8::/32"})
	if err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	size = matcher.GetBlocklistSize()
	if size != 4 {
		t.Errorf("Expected blocklist size 4, got %d", size)
	}

	// Test with comments and empty lines
	err = matcher.LoadBlocklist([]string{"# comment", "", "198.51.100.1", ""})
	if err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	size = matcher.GetBlocklistSize()
	if size != 1 {
		t.Errorf("Expected blocklist size 1 (comments and empty lines should be skipped), got %d", size)
	}
}

func TestIPMatcher_ClearWhitelist(t *testing.T) {
	matcher := NewIPMatcher()

	// Load whitelist
	err := matcher.LoadWhitelist([]string{"198.51.100.1", "203.0.113.0/24"})
	if err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	// Verify whitelist has entries
	size := matcher.GetWhitelistSize()
	if size != 2 {
		t.Errorf("Expected whitelist size 2, got %d", size)
	}

	// Test that whitelist works
	blocked, reason, err := matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to be whitelisted")
	}
	if reason != "whitelisted" {
		t.Errorf("Expected reason 'whitelisted', got: %s", reason)
	}

	// Clear whitelist
	matcher.ClearWhitelist()

	// Verify whitelist is empty
	size = matcher.GetWhitelistSize()
	if size != 0 {
		t.Errorf("Expected whitelist size 0 after clear, got %d", size)
	}

	// Test that IP is no longer whitelisted
	blocked, _, err = matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to not be blocked after whitelist clear (whitelist was cleared)")
	}
}

func TestIPMatcher_ClearBlocklist(t *testing.T) {
	matcher := NewIPMatcher()

	// Load blocklist
	err := matcher.LoadBlocklist([]string{"198.51.100.1", "203.0.113.0/24"})
	if err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	// Verify blocklist has entries
	size := matcher.GetBlocklistSize()
	if size != 2 {
		t.Errorf("Expected blocklist size 2, got %d", size)
	}

	// Test that blocklist works
	blocked, reason, err := matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !blocked {
		t.Error("Expected IP to be blocked")
	}
	if reason == "" {
		t.Error("Expected non-empty reason")
	}

	// Clear blocklist
	matcher.ClearBlocklist()

	// Verify blocklist is empty
	size = matcher.GetBlocklistSize()
	if size != 0 {
		t.Errorf("Expected blocklist size 0 after clear, got %d", size)
	}

	// Test that IP is no longer blocked
	blocked, _, err = matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to not be blocked after blocklist clear")
	}
}

func TestIPMatcher_ClearBoth(t *testing.T) {
	matcher := NewIPMatcher()

	// Load both whitelist and blocklist
	err := matcher.LoadWhitelist([]string{"198.51.100.1"})
	if err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	err = matcher.LoadBlocklist([]string{"198.51.100.2", "203.0.113.0/24"})
	if err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	// Verify both lists have entries
	whitelistSize := matcher.GetWhitelistSize()
	blocklistSize := matcher.GetBlocklistSize()
	if whitelistSize != 1 {
		t.Errorf("Expected whitelist size 1, got %d", whitelistSize)
	}
	if blocklistSize != 2 {
		t.Errorf("Expected blocklist size 2, got %d", blocklistSize)
	}

	// Test that whitelist takes precedence
	blocked, _, err := matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to be whitelisted")
	}

	// Test that blocklist works for other IPs
	blocked, _, err = matcher.IsBlocked("198.51.100.2")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !blocked {
		t.Error("Expected IP to be blocked")
	}

	// Clear both lists
	matcher.ClearWhitelist()
	matcher.ClearBlocklist()

	// Verify both lists are empty
	whitelistSize = matcher.GetWhitelistSize()
	blocklistSize = matcher.GetBlocklistSize()
	if whitelistSize != 0 {
		t.Errorf("Expected whitelist size 0 after clear, got %d", whitelistSize)
	}
	if blocklistSize != 0 {
		t.Errorf("Expected blocklist size 0 after clear, got %d", blocklistSize)
	}

	// Test that both IPs are no longer blocked or whitelisted
	blocked, _, err = matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP 198.51.100.1 to not be blocked after clear")
	}

	blocked, _, err = matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP 198.51.100.1 to not be blocked after clear")
	}
}

func TestIPMatcher_ThreadSafety(t *testing.T) {
	matcher := NewIPMatcher()

	// Load initial data
	err := matcher.LoadWhitelist([]string{"198.51.100.1"})
	if err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	err = matcher.LoadBlocklist([]string{"198.51.100.2", "203.0.113.0/24"})
	if err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	// Test concurrent reads
	done := make(chan bool, 2)

	// Goroutine 1: Read operations
	go func() {
		for i := 0; i < 1000; i++ {
			matcher.IsBlocked("198.51.100.2")
			matcher.IsBlocked("198.51.100.1")
			matcher.IsBlocked("203.0.113.2")
			_ = matcher.GetWhitelistSize()
			_ = matcher.GetBlocklistSize()
		}
		done <- true
	}()

	// Goroutine 2: Read operations
	go func() {
		for i := 0; i < 1000; i++ {
			matcher.IsBlocked("203.0.113.1")
			matcher.IsBlocked("2001:db8::1")
			_ = matcher.GetWhitelistSize()
			_ = matcher.GetBlocklistSize()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	for i := 0; i < 2; i++ {
		<-done
	}

	// Verify data integrity after concurrent reads
	blocked, _, err := matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to be whitelisted after concurrent reads")
	}

	blocked, _, err = matcher.IsBlocked("198.51.100.2")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !blocked {
		t.Error("Expected IP to be blocked after concurrent reads")
	}
}

func TestIPMatcher_InvalidEntries(t *testing.T) {
	matcher := NewIPMatcher()

	// Test loading whitelist with invalid entries
	err := matcher.LoadWhitelist([]string{"198.51.100.1", "invalid-ip", "300.168.1.1", "2001:db8::/129"})
	if err == nil {
		t.Error("Expected error for invalid IP entries in whitelist")
	}

	// Test loading blocklist with invalid entries
	err = matcher.LoadBlocklist([]string{"198.51.100.1", "invalid-ip", "300.168.1.1", "2001:db8::/129"})
	if err == nil {
		t.Error("Expected error for invalid IP entries in blocklist")
	}

	// Test loading whitelist with only valid entries
	err = matcher.LoadWhitelist([]string{"198.51.100.1", "203.0.113.0/24"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Test loading blocklist with only valid entries
	err = matcher.LoadBlocklist([]string{"172.16.0.1", "203.0.113.0/24"})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify valid entries work
	blocked, _, err := matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to be whitelisted")
	}

	blocked, _, err = matcher.IsBlocked("172.16.0.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !blocked {
		t.Error("Expected IP to be blocked")
	}
}

func TestIPMatcher_EmptyAndCommentLines(t *testing.T) {
	matcher := NewIPMatcher()

	// Test loading with empty lines and comments
	err := matcher.LoadWhitelist([]string{
		"",
		"# This is a comment",
		"",
		"198.51.100.1",
		"   ",
		"# Another comment",
		"203.0.113.0/24",
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify only valid entries are loaded
	size := matcher.GetWhitelistSize()
	if size != 2 {
		t.Errorf("Expected whitelist size 2 (empty lines and comments should be skipped), got %d", size)
	}

	// Test loading blocklist with empty lines and comments
	err = matcher.LoadBlocklist([]string{
		"",
		"# This is a comment",
		"",
		"172.16.0.1",
		"   ",
		"# Another comment",
		"203.0.113.0/24",
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify only valid entries are loaded
	size = matcher.GetBlocklistSize()
	if size != 2 {
		t.Errorf("Expected blocklist size 2 (empty lines and comments should be skipped), got %d", size)
	}

	// Test that valid entries work
	blocked, _, err := matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to be whitelisted")
	}

	blocked, _, err = matcher.IsBlocked("172.16.0.1")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !blocked {
		t.Error("Expected IP to be blocked")
	}
}
