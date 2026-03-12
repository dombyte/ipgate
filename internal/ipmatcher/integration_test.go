package ipmatcher

import (
	"testing"
)

// TestIntegration_BasicFlow tests a complete flow similar to how the system would use it
func TestIntegration_BasicFlow(t *testing.T) {
	// Create a new matcher
	matcher := NewIPMatcher()

	// Simulate loading whitelist and blocklist from files
	whitelistEntries := []string{
		"198.51.100.1",   // Specific IP
		"203.0.113.0/24", // CIDR range
		"2001:db8::1",    // IPv6 address
	}

	blocklistEntries := []string{
		"198.51.100.2",   // Specific IP
		"203.0.113.0/24", // CIDR range
		"2001:db8::2",    // IPv6 address
	}

	// Load the lists
	err := matcher.LoadWhitelist(whitelistEntries)
	if err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	err = matcher.LoadBlocklist(blocklistEntries)
	if err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	// Test cases
	testCases := []struct {
		ip          string
		blocked     bool
		reason      string
		description string
	}{
		{"198.51.100.1", false, "whitelisted", "Whitelisted specific IP"},
		{"203.0.113.1", false, "whitelisted", "Whitelisted via CIDR"},
		{"198.51.100.2", true, "matched IP 198.51.100.2", "Blocked IP"},
		{"198.51.100.3", false, "", "Neither whitelisted nor blocked"},
		{"203.0.113.2", false, "whitelisted", "Whitelisted via CIDR (whitelist takes precedence)"},
		{"198.51.100.2", true, "matched IP 198.51.100.2", "Blocked via CIDR"},
		{"2001:db8::1", false, "whitelisted", "Whitelisted IPv6"},
		{"2001:db8::2", true, "matched IP 2001:db8::2", "Blocked IPv6"},
		{"2001:db8::3", false, "", "Neither whitelisted nor blocked IPv6"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			blocked, reason, err := matcher.IsBlocked(tc.ip)
			if err != nil {
				t.Errorf("Unexpected error for %s: %v", tc.ip, err)
				return
			}

			if blocked != tc.blocked {
				t.Errorf("IP %s: expected blocked=%v, got blocked=%v", tc.ip, tc.blocked, blocked)
			}

			if reason != tc.reason {
				t.Errorf("IP %s: expected reason=%s, got reason=%s", tc.ip, tc.reason, reason)
			}
		})
	}
}

// TestIntegration_ConcurrentAccess tests concurrent access to the matcher
func TestIntegration_ConcurrentAccess(t *testing.T) {
	matcher := NewIPMatcher()

	// Load test data
	err := matcher.LoadWhitelist([]string{"198.51.100.1", "203.0.113.0/24"})
	if err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	err = matcher.LoadBlocklist([]string{"198.51.100.2", "203.0.113.0/24"})
	if err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	// Test concurrent access
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				ip := "192.168.1." + string(byte(j%256))
				_, _, _ = matcher.IsBlocked(ip)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify the matcher still works correctly after concurrent access
	blocked, _, err := matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Error after concurrent access: %v", err)
	}
	if blocked {
		t.Error("Expected whitelisted IP to remain whitelisted after concurrent access")
	}
}

// TestIntegration_Reload tests reloading the matcher with new data
func TestIntegration_Reload(t *testing.T) {
	matcher := NewIPMatcher()

	// Initial load
	err := matcher.LoadWhitelist([]string{"198.51.100.1"})
	if err != nil {
		t.Fatalf("Failed to load initial whitelist: %v", err)
	}

	err = matcher.LoadBlocklist([]string{"198.51.100.2"})
	if err != nil {
		t.Fatalf("Failed to load initial blocklist: %v", err)
	}

	// Verify initial state
	blocked, _, err := matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Error checking initial state: %v", err)
	}
	if blocked {
		t.Error("Expected 198.51.100.1 to be whitelisted initially")
	}

	// Reload with different data
	err = matcher.LoadWhitelist([]string{"198.51.100.3"})
	if err != nil {
		t.Fatalf("Failed to reload whitelist: %v", err)
	}

	err = matcher.LoadBlocklist([]string{"198.51.100.3"})
	if err != nil {
		t.Fatalf("Failed to reload blocklist: %v", err)
	}

	// Verify new state
	blocked, _, err = matcher.IsBlocked("198.51.100.1")
	if err != nil {
		t.Errorf("Error checking after reload: %v", err)
	}
	if blocked {
		t.Error("Expected 198.51.100.1 to not be blocked after reload (not in new lists)")
	}

	blocked, _, err = matcher.IsBlocked("198.51.100.3")
	if err != nil {
		t.Errorf("Error checking 198.51.100.3 after reload: %v", err)
	}
	if blocked {
		t.Error("Expected 198.51.100.3 to be whitelisted after reload (whitelist takes precedence)")
	}
}
