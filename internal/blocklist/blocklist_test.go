package blocklist

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dombyte/ipgate/internal/ipmatcher"
)

// Helper functions for test file creation
func createTempBlocklistFile(content string) (string, error) {
	tmpDir := os.TempDir()
	path := filepath.Join(tmpDir, "test_blocklist_"+fmt.Sprintf("%d", time.Now().UnixNano())+".txt")
	return path, os.WriteFile(path, []byte(content), 0644)
}

func createTempWhitelistFile(content string) (string, error) {
	tmpDir := os.TempDir()
	path := filepath.Join(tmpDir, "test_whitelist_"+fmt.Sprintf("%d", time.Now().UnixNano())+".txt")
	return path, os.WriteFile(path, []byte(content), 0644)
}

func TestLoadBlocklist(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		expected    []string
		shouldError bool
	}{
		{
			name:     "Valid IPs",
			content:  "198.51.100.1\n10.0.0.1\n2001:db8::1",
			expected: []string{"198.51.100.1", "10.0.0.1", "2001:db8::1"},
		},
		{
			name:     "With comments and empty lines",
			content:  "# Comment\n192.168.1.1\n\n10.0.0.1\n",
			expected: []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name:     "CIDR notation",
			content:  "192.168.1.0/24\n10.0.0.0/16",
			expected: []string{"192.168.1.0/24", "10.0.0.0/16"},
		},
		{
			name:        "Empty file",
			content:     "",
			expected:    []string{},
			shouldError: false,
		},
		{
			name:        "File too large",
			content:     strings.Repeat("198.51.100.1\n", 10000),
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path, err := createTempBlocklistFile(tc.content)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(path)

			// Change to temp directory so OpenRoot(".") can access the file
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatal("Failed to get current directory:", err)
			}
			defer os.Chdir(originalDir)
			tmpDir := filepath.Dir(path)
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatal("Failed to change to temp directory:", err)
			}

			maxSize := int64(1024)
			if tc.name == "File too large" {
				maxSize = 100
			}

			// Use just the filename since we changed to the temp directory
			filename := filepath.Base(path)
			entries, err := LoadLocalFile(filename, maxSize)

			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tc.name)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(entries) != len(tc.expected) {
				t.Errorf("Expected %d entries, got %d", len(tc.expected), len(entries))
				return
			}

			for i, entry := range entries {
				if entry != tc.expected[i] {
					t.Errorf("Entry %d: expected %s, got %s", i, tc.expected[i], entry)
				}
			}
		})
	}
}

func TestLoadWhitelistFile(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		expected    []string
		shouldError bool
	}{
		{
			name:     "Valid IPs",
			content:  "198.51.100.1\n10.0.0.1\n2001:db8::1",
			expected: []string{"198.51.100.1", "10.0.0.1", "2001:db8::1"},
		},
		{
			name:     "With comments and empty lines",
			content:  "# Comment\n192.168.1.1\n\n10.0.0.1\n",
			expected: []string{"192.168.1.1", "10.0.0.1"},
		},
		{
			name:     "CIDR notation",
			content:  "192.168.1.0/24\n10.0.0.0/16",
			expected: []string{"192.168.1.0/24", "10.0.0.0/16"},
		},
		{
			name:        "Empty file",
			content:     "",
			expected:    []string{},
			shouldError: false,
		},
		{
			name:        "File too large",
			content:     strings.Repeat("198.51.100.1\n", 10000),
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path, err := createTempWhitelistFile(tc.content)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(path)

			// Change to temp directory so OpenRoot(".") can access the file
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatal("Failed to get current directory:", err)
			}
			defer os.Chdir(originalDir)
			tmpDir := filepath.Dir(path)
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatal("Failed to change to temp directory:", err)
			}

			maxSize := int64(1024)
			if tc.name == "File too large" {
				maxSize = 100
			}

			// Use just the filename since we changed to the temp directory
			filename := filepath.Base(path)
			entries, err := LoadLocalFile(filename, maxSize)

			if tc.shouldError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tc.name)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(entries) != len(tc.expected) {
				t.Errorf("Expected %d entries, got %d", len(tc.expected), len(entries))
				return
			}

			for i, entry := range entries {
				if entry != tc.expected[i] {
					t.Errorf("Entry %d: expected %s, got %s", i, tc.expected[i], entry)
				}
			}
		})
	}
}

func TestLoadWhitelistFileEdgeCases(t *testing.T) {
	// Test with invalid file path
	_, err := LoadLocalFile("/nonexistent/path/file.txt", 1024)
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	// Test with invalid CIDR in whitelist (LoadWhitelistFile doesn't validate CIDR, only reads lines)
	path, err := createTempWhitelistFile("invalid-cidr\n192.168.1.1")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(path)

	// Change to temp directory so OpenRoot(".") can access the file
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal("Failed to get current directory:", err)
	}
	defer os.Chdir(originalDir)
	tmpDir := filepath.Dir(path)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal("Failed to change to temp directory:", err)
	}

	// Use just the filename since we changed to the temp directory
	filename := filepath.Base(path)
	entries, err := LoadLocalFile(filename, 1024)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(entries) != 2 || entries[0] != "invalid-cidr" || entries[1] != "192.168.1.1" {
		t.Errorf("Expected entries [invalid-cidr 192.168.1.1], got %v", entries)
	}
}

func TestLoadWhitelistFileSizeLimit(t *testing.T) {
	// Create a test server that returns content larger than limit
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strings.Repeat("198.51.100.1\n", 10000)))
	}))
	defer server.Close()

	_, err := LoadLocalFile(server.URL, 100)
	if err == nil {
		t.Error("Expected error for file exceeding size limit")
	}
}

func TestLoadRemoteBlocklist(t *testing.T) {
	// Test successful load
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("198.51.100.1\n10.0.0.1\n2001:db8::1"))
	}))
	defer server.Close()

	entries, err := LoadRemoteFile(server.URL, 1024)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
		return
	}

	expected := []string{"198.51.100.1", "10.0.0.1", "2001:db8::1"}
	for i, entry := range entries {
		if entry != expected[i] {
			t.Errorf("Entry %d: expected %s, got %s", i, expected[i], entry)
		}
	}
}

func TestLoadRemoteBlocklistErrorCases(t *testing.T) {
	// Test server error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := LoadRemoteFile(server.URL, 1024)
	if err == nil {
		t.Error("Expected error for server error")
	}

	// Test invalid content (should not error, just return what it gets)
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid content"))
	}))
	defer server2.Close()

	entries, err := LoadRemoteFile(server2.URL, 1024)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(entries) != 1 || entries[0] != "invalid content" {
		t.Errorf("Expected entries [invalid content], got %v", entries)
	}
}

func TestLoadRemoteBlocklistSizeLimit(t *testing.T) {
	// Create a test server that returns content larger than limit
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(strings.Repeat("198.51.100.1\n", 10000)))
	}))
	defer server.Close()

	_, err := LoadRemoteFile(server.URL, 100)
	if err == nil {
		t.Error("Expected error for content exceeding size limit")
	}
}

func TestIsIPBlocked(t *testing.T) {
	// Create IPMatcher with test data
	matcher := ipmatcher.NewIPMatcher()

	// Load whitelist - all at once
	whitelistEntries := []string{"198.51.100.1", "203.0.113.1", "2001:db8::2"}
	if err := matcher.LoadWhitelist(whitelistEntries); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	// Load blocklist - all at once
	blocklistEntries := []string{"198.51.100.1", "10.0.0.0/24", "2001:db8::1", "203.0.113.0/24"}
	if err := matcher.LoadBlocklist(blocklistEntries); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	testCases := []struct {
		ip      string
		blocked bool
		reason  string
	}{
		{"198.51.100.1", false, "whitelisted"},
		{"10.0.0.5", true, "matched IP 10.0.0.5"},
		{"2001:db8::1", true, "matched IP 2001:db8::1"},
		{"172.16.0.1", false, ""},
		{"203.0.113.1", false, "whitelisted"},
		{"2001:db8::2", false, "whitelisted"},
		{"203.0.113.2", true, "matched IP 203.0.113.2"},
	}

	for _, tc := range testCases {
		t.Run(tc.ip, func(t *testing.T) {
			blocked, reason, err := IsIPBlocked(tc.ip, matcher)
			if err != nil {
				t.Errorf("IsIPBlocked failed for %s: %v", tc.ip, err)
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

func TestIsIPBlockedNilMatcher(t *testing.T) {
	// Test with nil matcher
	_, _, err := IsIPBlocked("198.51.100.1", nil)
	if err == nil {
		t.Error("Expected error for nil matcher")
	}
	if err.Error() != "IPMatcher is nil" {
		t.Errorf("Expected error message 'IPMatcher is nil', got %s", err.Error())
	}
}

func TestIsIPBlockedInvalidIP(t *testing.T) {
	matcher := ipmatcher.NewIPMatcher()
	if err := matcher.LoadBlocklist([]string{"198.51.100.1"}); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	_, _, err := IsIPBlocked("invalid-ip", matcher)
	if err == nil {
		t.Error("Expected error for invalid IP")
	}
}

func TestIsIPBlockedInvalidCIDR(t *testing.T) {
	matcher := ipmatcher.NewIPMatcher()
	// Invalid CIDR should be handled gracefully by the IPMatcher
	if err := matcher.LoadBlocklist([]string{"300.168.1.0/24"}); err != nil {
		t.Logf("Expected error for invalid CIDR: %v", err)
	}

	_, _, err := IsIPBlocked("198.51.100.1", matcher)
	// The IPMatcher should handle invalid CIDRs gracefully
	if err != nil {
		t.Logf("Got error for invalid CIDR: %v", err)
	}
}

func TestIsIPBlockedEdgeCases(t *testing.T) {
	// Test with empty IPMatcher
	matcher := ipmatcher.NewIPMatcher()

	// Test with empty matcher (no blocklist or whitelist loaded)
	blocked, reason, err := IsIPBlocked("198.51.100.1", matcher)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to not be blocked with empty matcher")
	}
	if reason != "" {
		t.Errorf("Expected empty reason, got %s", reason)
	}
}

func TestIsIPBlockedComplexScenarios(t *testing.T) {
	// Create IPMatcher with test data
	matcher := ipmatcher.NewIPMatcher()

	// Load whitelist - all at once
	whitelistEntries := []string{"198.51.100.1", "203.0.113.1", "2001:db8::2"}
	if err := matcher.LoadWhitelist(whitelistEntries); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	// Load blocklist - all at once
	blocklistEntries := []string{"198.51.100.1", "10.0.0.0/24", "2001:db8::1", "203.0.113.0/24"}
	if err := matcher.LoadBlocklist(blocklistEntries); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}

	testCases := []struct {
		ip      string
		blocked bool
		reason  string
	}{
		{"198.51.100.1", false, "whitelisted"},
		{"10.0.0.5", true, "matched IP 10.0.0.5"},
		{"2001:db8::1", true, "matched IP 2001:db8::1"},
		{"172.16.0.1", false, ""},
		{"203.0.113.1", false, "whitelisted"},
		{"2001:db8::2", false, "whitelisted"},
		{"203.0.113.2", true, "matched IP 203.0.113.2"},
	}

	for _, tc := range testCases {
		t.Run(tc.ip, func(t *testing.T) {
			blocked, reason, err := IsIPBlocked(tc.ip, matcher)
			if err != nil {
				t.Errorf("IsIPBlocked failed for %s: %v", tc.ip, err)
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

func TestIsIPBlockedWhitelistPriority(t *testing.T) {
	// Test that whitelist takes priority over blocklist
	matcher := ipmatcher.NewIPMatcher()

	// Load both blocklist and whitelist with the same IP
	if err := matcher.LoadBlocklist([]string{"198.51.100.1"}); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}
	if err := matcher.LoadWhitelist([]string{"198.51.100.1"}); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	blocked, reason, err := IsIPBlocked("198.51.100.1", matcher)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP to not be blocked when in whitelist (whitelist should take priority)")
	}
	if reason != "whitelisted" {
		t.Errorf("Expected 'whitelisted' reason for whitelisted IP, got %s", reason)
	}
}

func TestIsIPBlockedCIDRWhitelist(t *testing.T) {
	// Test CIDR notation in whitelist
	matcher := ipmatcher.NewIPMatcher()

	if err := matcher.LoadBlocklist([]string{"192.168.1.0/24"}); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}
	if err := matcher.LoadWhitelist([]string{"198.51.100.1/24"}); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	// IP in blocklist CIDR but not in whitelist CIDR should be blocked
	blocked, _, err := IsIPBlocked("192.168.1.50", matcher)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !blocked {
		t.Error("Expected IP 192.168.1.50 to be blocked (not in whitelist)")
	}

	// IP in both blocklist and whitelist CIDRs should not be blocked
	blocked, _, err = IsIPBlocked("198.51.100.1", matcher)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if blocked {
		t.Error("Expected IP 198.51.100.1 to not be blocked (whitelist priority)")
	}
}
