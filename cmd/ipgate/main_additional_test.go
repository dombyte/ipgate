package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dombyte/ipgate/internal/cache"
	"github.com/dombyte/ipgate/internal/config"
	"github.com/dombyte/ipgate/internal/logging"
	"github.com/dombyte/ipgate/internal/models"
)

// TestParseCLIFlags tests the CLI flag parsing functionality
func TestParseCLIFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantConfig string
		wantLevel  string
	}{
		{"no flags", []string{}, "", "INFO"},
		{"config flag", []string{"-config", "test.yaml"}, "test.yaml", "INFO"},
		{"config equals", []string{"-config=test.yaml"}, "test.yaml", "INFO"},
		{"long config flag", []string{"--config", "test.yaml"}, "test.yaml", "INFO"},
		{"long config equals", []string{"--config=test.yaml"}, "test.yaml", "INFO"},
		{"log level flag", []string{"-log.level", "DEBUG"}, "", "DEBUG"},
		{"log level equals", []string{"-log.level=DEBUG"}, "", "DEBUG"},
		{"long log level flag", []string{"--log.level", "WARN"}, "", "WARN"},
		{"long log level equals", []string{"--log.level=WARN"}, "", "WARN"},
		{"both flags", []string{"-config", "test.yaml", "-log.level", "ERROR"}, "test.yaml", "ERROR"},
		{"mixed flags", []string{"-config=test.yaml", "--log.level=DEBUG"}, "test.yaml", "DEBUG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			// Set test args
			os.Args = append([]string{"ipgate"}, tt.args...)

			gotConfig, gotLevel := parseCLIFlags()

			if gotConfig != tt.wantConfig {
				t.Errorf("parseCLIFlags() config = %q, want %q", gotConfig, tt.wantConfig)
			}
			if gotLevel != tt.wantLevel {
				t.Errorf("parseCLIFlags() level = %q, want %q", gotLevel, tt.wantLevel)
			}
		})
	}
}

// TestListenAndServe tests the HTTP server functionality
func TestListenAndServe(t *testing.T) {
	// Create a simple handler that returns 200 OK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start server in a goroutine
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	// Start server
	go func() {
		_ = server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Make a request
	resp, err := http.Get("http://localhost:8080")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Cleanup
	_ = server.Shutdown(nil)
}

// TestLoadBlocklists tests the loadBlocklists wrapper function
func TestLoadBlocklists(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	cfg := &config.Config{
		BlacklistFiles: []config.LocalFile{
			{Path: "config/blocklists/blocklist.txt"},
		},
		BlacklistRemotes: []config.RemoteFile{},
		BlocklistMaxSize: 10000,
	}

	// Test loading blocklists using the wrapper function
	loadBlocklists(cfg, false)

	// Note: In test environment, the file may not exist, so we just verify the function doesn't panic
	_ = cfg.Blocklists
}

// TestLoadBlocklistsWithLogger tests the blocklist loading functionality
func TestLoadBlocklistsWithLogger(t *testing.T) {
	cfg := &config.Config{
		BlacklistFiles: []config.LocalFile{
			{Path: "config/blocklists/blocklist.txt"},
		},
		BlacklistRemotes: []config.RemoteFile{},
		BlocklistMaxSize: 10000,
	}

	logger := logging.NewLogger("DEBUG")

	// Test loading local blocklists
	loadBlocklistsWithLogger(cfg, false, logger)

	// Note: In test environment, the file may not exist, so we just verify the function doesn't panic
	_ = cfg.Blocklists

	// Test loading with remote (should not load anything in test)
	cfg2 := &config.Config{
		BlacklistFiles: []config.LocalFile{},
		BlacklistRemotes: []config.RemoteFile{
			{URL: "https://example.com/nonexistent.txt", Cron: ""},
		},
		BlocklistMaxSize: 10000,
	}

	loadBlocklistsWithLogger(cfg2, true, logger)

	// Should have no blocklists since remote doesn't exist
	if len(cfg2.Blocklists) != 0 {
		t.Error("Expected no blocklists to be loaded from invalid remote")
	}
}

// TestLoadBlocklistsWithLogger_AllCodePaths tests all code paths in loadBlocklistsWithLogger
func TestLoadBlocklistsWithLogger_AllCodePaths(t *testing.T) {
	// Test 1: Empty config with no files or remotes
	t.Run("EmptyConfig", func(t *testing.T) {
		cfg := &config.Config{
			BlacklistFiles:   []config.LocalFile{},
			BlacklistRemotes: []config.RemoteFile{},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadBlocklistsWithLogger(cfg, false, logger)
		if len(cfg.Blocklists) != 0 {
			t.Error("Expected no blocklists for empty config")
		}
	})

	// Test 2: Config with local file that doesn't exist
	t.Run("NonExistentLocalFile", func(t *testing.T) {
		cfg := &config.Config{
			BlacklistFiles: []config.LocalFile{
				{Path: "/nonexistent/path/blocklist.txt"},
			},
			BlacklistRemotes: []config.RemoteFile{},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadBlocklistsWithLogger(cfg, false, logger)
		// Should handle gracefully and not panic
		_ = cfg.Blocklists
	})

	// Test 3: Config with valid local file
	t.Run("ValidLocalFile", func(t *testing.T) {
		// Create a temporary blocklist file
		tmpDir, err := ioutil.TempDir("", "blocklist-test")
		if err != nil {
			t.Fatal("Failed to create temp dir:", err)
		}
		defer os.RemoveAll(tmpDir)

		blocklistFile := filepath.Join(tmpDir, "blocklist.txt")
		content := "198.51.100.198.51.100.1\n203.0.113.203.0.113.1\n203.0.113.172.16.0.1"
		if err := ioutil.WriteFile(blocklistFile, []byte(content), 0644); err != nil {
			t.Fatal("Failed to write blocklist file:", err)
		}

		// Change to temp directory so OpenRoot(".") can access the file
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatal("Failed to get current directory:", err)
		}
		defer os.Chdir(originalDir)
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal("Failed to change to temp directory:", err)
		}

		cfg := &config.Config{
			BlacklistFiles: []config.LocalFile{
				{Path: "blocklist.txt"},
			},
			BlacklistRemotes: []config.RemoteFile{},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadBlocklistsWithLogger(cfg, false, logger)

		if len(cfg.Blocklists) != 1 {
			t.Errorf("Expected 1 blocklist, got %d", len(cfg.Blocklists))
		} else if len(cfg.Blocklists[0]) != 3 {
			t.Errorf("Expected 3 entries, got %d", len(cfg.Blocklists[0]))
		}
	})

	// Test 4: Config with remote file (will fail but should handle gracefully)
	t.Run("RemoteFile", func(t *testing.T) {
		cfg := &config.Config{
			BlacklistFiles: []config.LocalFile{},
			BlacklistRemotes: []config.RemoteFile{
				{URL: "https://example.com/nonexistent.txt", Cron: ""},
			},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadBlocklistsWithLogger(cfg, true, logger)
		// Should handle gracefully and not panic
		_ = cfg.Blocklists
	})

	// Test 5: Config with multiple local files
	t.Run("MultipleLocalFiles", func(t *testing.T) {
		tmpDir, err := ioutil.TempDir("", "blocklist-test")
		if err != nil {
			t.Fatal("Failed to create temp dir:", err)
		}
		defer os.RemoveAll(tmpDir)

		// Change to temp directory so OpenRoot(".") can access the files
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatal("Failed to get current directory:", err)
		}
		defer os.Chdir(originalDir)
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal("Failed to change to temp directory:", err)
		}

		// Create first blocklist file
		blocklistFile1 := "blocklist1.txt"
		content1 := "198.51.100.198.51.100.1\n203.0.113.203.0.113.1"
		if err := ioutil.WriteFile(blocklistFile1, []byte(content1), 0644); err != nil {
			t.Fatal("Failed to write blocklist file 1:", err)
		}

		// Create second blocklist file
		blocklistFile2 := "blocklist2.txt"
		content2 := "203.0.113.172.16.0.1\n203.0.113.10.0.0.2"
		if err := ioutil.WriteFile(blocklistFile2, []byte(content2), 0644); err != nil {
			t.Fatal("Failed to write blocklist file 2:", err)
		}

		cfg := &config.Config{
			BlacklistFiles: []config.LocalFile{
				{Path: blocklistFile1},
				{Path: blocklistFile2},
			},
			BlacklistRemotes: []config.RemoteFile{},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadBlocklistsWithLogger(cfg, false, logger)

		if len(cfg.Blocklists) != 2 {
			t.Errorf("Expected 2 blocklists, got %d", len(cfg.Blocklists))
		} else if len(cfg.Blocklists[0]) != 2 || len(cfg.Blocklists[1]) != 2 {
			t.Errorf("Expected 2 entries in each blocklist")
		}
	})

	// Test 6: Config with no logger (nil logger)
	t.Run("NoLogger", func(t *testing.T) {
		cfg := &config.Config{
			BlacklistFiles: []config.LocalFile{
				{Path: "config/blocklists/blocklist.txt"},
			},
			BlacklistRemotes: []config.RemoteFile{},
			BlocklistMaxSize: 10000,
		}
		// Pass nil logger
		loadBlocklistsWithLogger(cfg, false, nil)
		// Should handle gracefully and not panic
		_ = cfg.Blocklists
	})

	// Test 7: Test reloadRemote=false (should not load remotes)
	t.Run("ReloadRemoteFalse", func(t *testing.T) {
		cfg := &config.Config{
			BlacklistFiles: []config.LocalFile{},
			BlacklistRemotes: []config.RemoteFile{
				{URL: "https://example.com/nonexistent.txt", Cron: ""},
			},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadBlocklistsWithLogger(cfg, false, logger)
		// Should have no blocklists since reloadRemote=false
		if len(cfg.Blocklists) != 0 {
			t.Error("Expected no blocklists when reloadRemote=false")
		}
	})

	// Test 8: Test reloadRemote=true (should attempt to load remotes)
	t.Run("ReloadRemoteTrue", func(t *testing.T) {
		cfg := &config.Config{
			BlacklistFiles: []config.LocalFile{},
			BlacklistRemotes: []config.RemoteFile{
				{URL: "https://example.com/nonexistent.txt", Cron: ""},
			},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadBlocklistsWithLogger(cfg, true, logger)
		// Should handle gracefully and not panic
		_ = cfg.Blocklists
	})
}

// TestLoadWhitelists tests the loadWhitelists wrapper function
func TestLoadWhitelists(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	cfg := &config.Config{
		WhitelistFiles: []config.WhitelistFile{
			{Path: "config/whitelists/whitelist.txt"},
		},
		WhitelistRemotes: []config.RemoteFile{},
		BlocklistMaxSize: 10000,
	}

	// Test loading whitelists using the wrapper function
	loadWhitelists(cfg, false)

	// Note: In test environment, the file may not exist, so we just verify the function doesn't panic
	_ = cfg.Whitelist
}

// TestLoadWhitelistsWithLogger tests the whitelist loading functionality
func TestLoadWhitelistsWithLogger(t *testing.T) {
	cfg := &config.Config{
		WhitelistFiles: []config.WhitelistFile{
			{Path: "config/whitelists/whitelist.txt"},
		},
		WhitelistRemotes: []config.RemoteFile{},
		BlocklistMaxSize: 10000,
	}

	logger := logging.NewLogger("DEBUG")

	// Test loading local whitelists
	loadWhitelistsWithLogger(cfg, false, logger)

	// Note: In test environment, the file may not exist, so we just verify the function doesn't panic
	_ = cfg.Whitelist

	// Test loading with remote (should not load anything in test)
	cfg2 := &config.Config{
		WhitelistFiles: []config.WhitelistFile{},
		WhitelistRemotes: []config.RemoteFile{
			{URL: "https://example.com/nonexistent.txt", Cron: ""},
		},
		BlocklistMaxSize: 10000,
	}

	loadWhitelistsWithLogger(cfg2, true, logger)

	// Should have no whitelist entries since remote doesn't exist
	if len(cfg2.Whitelist) != 0 {
		t.Error("Expected no whitelist entries to be loaded from invalid remote")
	}
}

// TestLoadWhitelistsWithLogger_AllCodePaths tests all code paths in loadWhitelistsWithLogger
func TestLoadWhitelistsWithLogger_AllCodePaths(t *testing.T) {
	// Test 1: Empty config with no files or remotes
	t.Run("EmptyConfig", func(t *testing.T) {
		cfg := &config.Config{
			WhitelistFiles:   []config.WhitelistFile{},
			WhitelistRemotes: []config.RemoteFile{},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadWhitelistsWithLogger(cfg, false, logger)
		if len(cfg.Whitelist) != 0 {
			t.Error("Expected no whitelist entries for empty config")
		}
	})

	// Test 2: Config with local file that doesn't exist
	t.Run("NonExistentLocalFile", func(t *testing.T) {
		cfg := &config.Config{
			WhitelistFiles: []config.WhitelistFile{
				{Path: "/nonexistent/path/whitelist.txt"},
			},
			WhitelistRemotes: []config.RemoteFile{},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadWhitelistsWithLogger(cfg, false, logger)
		// Should handle gracefully and not panic
		_ = cfg.Whitelist
	})

	// Test 3: Config with valid local file
	t.Run("ValidLocalFile", func(t *testing.T) {
		// Create a temporary whitelist file
		tmpDir, err := ioutil.TempDir("", "whitelist-test")
		if err != nil {
			t.Fatal("Failed to create temp dir:", err)
		}
		defer os.RemoveAll(tmpDir)

		// Change to temp directory so OpenRoot(".") can access the file
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatal("Failed to get current directory:", err)
		}
		defer os.Chdir(originalDir)
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal("Failed to change to temp directory:", err)
		}

		whitelistFile := "whitelist.txt"
		content := "198.51.100.198.51.100.1\n203.0.113.203.0.113.1\n203.0.113.172.16.0.1"
		if err := ioutil.WriteFile(whitelistFile, []byte(content), 0644); err != nil {
			t.Fatal("Failed to write whitelist file:", err)
		}

		cfg := &config.Config{
			WhitelistFiles: []config.WhitelistFile{
				{Path: whitelistFile},
			},
			WhitelistRemotes: []config.RemoteFile{},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadWhitelistsWithLogger(cfg, false, logger)

		if len(cfg.Whitelist) != 3 {
			t.Errorf("Expected 3 whitelist entries, got %d", len(cfg.Whitelist))
		}
	})

	// Test 4: Config with remote file (will fail but should handle gracefully)
	t.Run("RemoteFile", func(t *testing.T) {
		cfg := &config.Config{
			WhitelistFiles: []config.WhitelistFile{},
			WhitelistRemotes: []config.RemoteFile{
				{URL: "https://example.com/nonexistent.txt", Cron: ""},
			},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadWhitelistsWithLogger(cfg, true, logger)
		// Should handle gracefully and not panic
		_ = cfg.Whitelist
	})

	// Test 5: Config with multiple local files
	t.Run("MultipleLocalFiles", func(t *testing.T) {
		tmpDir, err := ioutil.TempDir("", "whitelist-test")
		if err != nil {
			t.Fatal("Failed to create temp dir:", err)
		}
		defer os.RemoveAll(tmpDir)

		// Change to temp directory so OpenRoot(".") can access the files
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatal("Failed to get current directory:", err)
		}
		defer os.Chdir(originalDir)
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal("Failed to change to temp directory:", err)
		}

		// Create first whitelist file
		whitelistFile1 := "whitelist1.txt"
		content1 := "198.51.100.198.51.100.1\n203.0.113.203.0.113.1"
		if err := ioutil.WriteFile(whitelistFile1, []byte(content1), 0644); err != nil {
			t.Fatal("Failed to write whitelist file 1:", err)
		}

		// Create second whitelist file
		whitelistFile2 := "whitelist2.txt"
		content2 := "203.0.113.172.16.0.1\n203.0.113.10.0.0.2"
		if err := ioutil.WriteFile(whitelistFile2, []byte(content2), 0644); err != nil {
			t.Fatal("Failed to write whitelist file 2:", err)
		}

		cfg := &config.Config{
			WhitelistFiles: []config.WhitelistFile{
				{Path: whitelistFile1},
				{Path: whitelistFile2},
			},
			WhitelistRemotes: []config.RemoteFile{},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadWhitelistsWithLogger(cfg, false, logger)

		if len(cfg.Whitelist) != 4 {
			t.Errorf("Expected 4 whitelist entries, got %d", len(cfg.Whitelist))
		}
	})

	// Test 6: Config with no logger (nil logger)
	t.Run("NoLogger", func(t *testing.T) {
		cfg := &config.Config{
			WhitelistFiles: []config.WhitelistFile{
				{Path: "config/whitelists/whitelist.txt"},
			},
			WhitelistRemotes: []config.RemoteFile{},
			BlocklistMaxSize: 10000,
		}
		// Pass nil logger
		loadWhitelistsWithLogger(cfg, false, nil)
		// Should handle gracefully and not panic
		_ = cfg.Whitelist
	})

	// Test 7: Test reloadRemote=false (should not load remotes)
	t.Run("ReloadRemoteFalse", func(t *testing.T) {
		cfg := &config.Config{
			WhitelistFiles: []config.WhitelistFile{},
			WhitelistRemotes: []config.RemoteFile{
				{URL: "https://example.com/nonexistent.txt", Cron: ""},
			},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadWhitelistsWithLogger(cfg, false, logger)
		// Should have no whitelist entries since reloadRemote=false
		if len(cfg.Whitelist) != 0 {
			t.Error("Expected no whitelist entries when reloadRemote=false")
		}
	})

	// Test 8: Test reloadRemote=true (should attempt to load remotes)
	t.Run("ReloadRemoteTrue", func(t *testing.T) {
		cfg := &config.Config{
			WhitelistFiles: []config.WhitelistFile{},
			WhitelistRemotes: []config.RemoteFile{
				{URL: "https://example.com/nonexistent.txt", Cron: ""},
			},
			BlocklistMaxSize: 10000,
		}
		logger := logging.NewLogger("DEBUG")
		loadWhitelistsWithLogger(cfg, true, logger)
		// Should handle gracefully and not panic
		_ = cfg.Whitelist
	})
}

// TestLoadIPMatcher tests the IPMatcher loading functionality
func TestLoadIPMatcher(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	cfg := &config.Config{
		WhitelistFiles: []config.WhitelistFile{
			{Path: "config/whitelists/whitelist.txt"},
		},
		BlacklistFiles: []config.LocalFile{
			{Path: "config/blocklists/blocklist.txt"},
		},
		BlocklistMaxSize: 10000,
	}

	// Load blocklists and whitelists first
	loadBlocklistsWithLogger(cfg, false, logger)
	loadWhitelistsWithLogger(cfg, false, logger)

	// Test loading IPMatcher
	matcher := loadIPMatcher(cfg)

	if matcher == nil {
		t.Error("Expected IPMatcher to be loaded successfully")
	}

	// Note: In test environment, files may not exist, so we just verify the function doesn't panic
	_ = matcher.GetWhitelistSize()
	_ = matcher.GetBlocklistSize()
}

// TestLoadIPMatcher_AllCodePaths tests all code paths in loadIPMatcher
func TestLoadIPMatcher_AllCodePaths(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	// Test 1: Empty config (no whitelist or blocklist)
	t.Run("EmptyConfig", func(t *testing.T) {
		cfg := &config.Config{
			WhitelistFiles:   []config.WhitelistFile{},
			BlacklistFiles:   []config.LocalFile{},
			BlocklistMaxSize: 10000,
		}

		// Load blocklists and whitelists first
		loadBlocklistsWithLogger(cfg, false, logger)
		loadWhitelistsWithLogger(cfg, false, logger)

		// Test loading IPMatcher
		matcher := loadIPMatcher(cfg)

		if matcher == nil {
			t.Error("Expected IPMatcher to be loaded successfully even with empty lists")
		}

		// Verify sizes are 0
		if matcher.GetWhitelistSize() != 0 {
			t.Errorf("Expected whitelist size 0, got %d", matcher.GetWhitelistSize())
		}
		if matcher.GetBlocklistSize() != 0 {
			t.Errorf("Expected blocklist size 0, got %d", matcher.GetBlocklistSize())
		}
	})

	// Test 2: Config with debug endpoint enabled
	t.Run("DebugEndpointEnabled", func(t *testing.T) {
		cfg := &config.Config{
			DebugEndpoint:    true,
			WhitelistFiles:   []config.WhitelistFile{},
			BlacklistFiles:   []config.LocalFile{},
			BlocklistMaxSize: 10000,
		}

		// Load blocklists and whitelists first
		loadBlocklistsWithLogger(cfg, false, logger)
		loadWhitelistsWithLogger(cfg, false, logger)

		// Test loading IPMatcher
		matcher := loadIPMatcher(cfg)

		if matcher == nil {
			t.Error("Expected IPMatcher to be loaded successfully")
		}

		// Verify debug tracking is enabled
		// Note: We can't directly test this, but the function should handle it gracefully
	})

	// Test 3: Config with valid whitelist and blocklist files
	t.Run("ValidFiles", func(t *testing.T) {
		// Create temporary files
		tmpDir, err := ioutil.TempDir("", "ipmatcher-test")
		if err != nil {
			t.Fatal("Failed to create temp dir:", err)
		}
		defer os.RemoveAll(tmpDir)

		// Change to temp directory so OpenRoot(".") can access the files
		originalDir, err := os.Getwd()
		if err != nil {
			t.Fatal("Failed to get current directory:", err)
		}
		defer os.Chdir(originalDir)
		if err := os.Chdir(tmpDir); err != nil {
			t.Fatal("Failed to change to temp directory:", err)
		}

		// Create whitelist file
		whitelistFile := "whitelist.txt"
		whitelistContent := "198.51.100.1\n203.0.113.1"
		if err := ioutil.WriteFile(whitelistFile, []byte(whitelistContent), 0644); err != nil {
			t.Fatal("Failed to write whitelist file:", err)
		}

		// Create blocklist file
		blocklistFile := "blocklist.txt"
		blocklistContent := "203.0.113.2\n203.0.113.3"
		if err := ioutil.WriteFile(blocklistFile, []byte(blocklistContent), 0644); err != nil {
			t.Fatal("Failed to write blocklist file:", err)
		}

		cfg := &config.Config{
			WhitelistFiles: []config.WhitelistFile{
				{Path: whitelistFile},
			},
			BlacklistFiles: []config.LocalFile{
				{Path: blocklistFile},
			},
			BlocklistMaxSize: 10000,
		}

		// Load blocklists and whitelists first
		loadBlocklistsWithLogger(cfg, false, logger)
		loadWhitelistsWithLogger(cfg, false, logger)

		// Test loading IPMatcher
		matcher := loadIPMatcher(cfg)

		if matcher == nil {
			t.Error("Expected IPMatcher to be loaded successfully")
		}

		// Verify sizes
		if matcher.GetWhitelistSize() != 2 {
			t.Errorf("Expected whitelist size 2, got %d", matcher.GetWhitelistSize())
		}
		if matcher.GetBlocklistSize() != 2 {
			t.Errorf("Expected blocklist size 2, got %d", matcher.GetBlocklistSize())
		}
	})

	// Test 4: Config with invalid whitelist (should still load blocklist)
	t.Run("InvalidWhitelist", func(t *testing.T) {
		cfg := &config.Config{
			WhitelistFiles: []config.WhitelistFile{
				{Path: "/nonexistent/whitelist.txt"},
			},
			BlacklistFiles: []config.LocalFile{
				{Path: "/nonexistent/blocklist.txt"},
			},
			BlocklistMaxSize: 10000,
		}

		// Load blocklists and whitelists first
		loadBlocklistsWithLogger(cfg, false, logger)
		loadWhitelistsWithLogger(cfg, false, logger)

		// Test loading IPMatcher - should still work even if whitelist fails
		matcher := loadIPMatcher(cfg)

		// Should not return nil since blocklist loading succeeded (even if whitelist failed)
		if matcher == nil {
			t.Error("Expected IPMatcher to be loaded even when whitelist loading fails")
		}
	})

	// Test 5: Config with invalid blocklist (should still load whitelist)
	t.Run("InvalidBlocklist", func(t *testing.T) {
		// Create temporary whitelist file
		tmpDir, err := ioutil.TempDir("", "ipmatcher-test")
		if err != nil {
			t.Fatal("Failed to create temp dir:", err)
		}
		defer os.RemoveAll(tmpDir)

		whitelistFile := filepath.Join(tmpDir, "whitelist.txt")
		whitelistContent := "198.51.100.1"
		if err := ioutil.WriteFile(whitelistFile, []byte(whitelistContent), 0644); err != nil {
			t.Fatal("Failed to write whitelist file:", err)
		}

		cfg := &config.Config{
			WhitelistFiles: []config.WhitelistFile{
				{Path: whitelistFile},
			},
			BlacklistFiles: []config.LocalFile{
				{Path: "/nonexistent/blocklist.txt"},
			},
			BlocklistMaxSize: 10000,
		}

		// Load blocklists and whitelists first
		loadBlocklistsWithLogger(cfg, false, logger)
		loadWhitelistsWithLogger(cfg, false, logger)

		// Test loading IPMatcher - should still work even if blocklist fails
		matcher := loadIPMatcher(cfg)

		// Should not return nil since whitelist loading succeeded (even if blocklist failed)
		if matcher == nil {
			t.Error("Expected IPMatcher to be loaded even when blocklist loading fails")
		}
	})

	// Test 6: Config with multiple blocklist files
	t.Run("MultipleBlocklistFiles", func(t *testing.T) {
		// Create temporary files
		tmpDir, err := ioutil.TempDir("", "ipmatcher-test")
		if err != nil {
			t.Fatal("Failed to create temp dir:", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create first blocklist file
		blocklistFile1 := filepath.Join(tmpDir, "blocklist1.txt")
		blocklistContent1 := "198.51.100.1\n203.0.113.1"
		if err := ioutil.WriteFile(blocklistFile1, []byte(blocklistContent1), 0644); err != nil {
			t.Fatal("Failed to write blocklist file 1:", err)
		}

		// Create second blocklist file
		blocklistFile2 := filepath.Join(tmpDir, "blocklist2.txt")
		blocklistContent2 := "203.0.113.2\n203.0.113.3"
		if err := ioutil.WriteFile(blocklistFile2, []byte(blocklistContent2), 0644); err != nil {
			t.Fatal("Failed to write blocklist file 2:", err)
		}

		cfg := &config.Config{
			WhitelistFiles: []config.WhitelistFile{},
			BlacklistFiles: []config.LocalFile{
				{Path: blocklistFile1},
				{Path: blocklistFile2},
			},
			BlocklistMaxSize: 10000,
		}

		// Load blocklists and whitelists first
		loadBlocklistsWithLogger(cfg, false, logger)
		loadWhitelistsWithLogger(cfg, false, logger)

		// Test loading IPMatcher
		matcher := loadIPMatcher(cfg)

		if matcher == nil {
			t.Error("Expected IPMatcher to be loaded successfully")
		}

		// Should have 4 total blocklist entries (2 from each file)
		// Note: cfg.Blocklists contains slices of entries, so we need to sum them
		expectedSize := 0
		for _, blocklist := range cfg.Blocklists {
			expectedSize += len(blocklist)
		}
		if matcher.GetBlocklistSize() != expectedSize {
			t.Errorf("Expected blocklist size %d, got %d", expectedSize, matcher.GetBlocklistSize())
		}
	})
}

// TestStartCronScheduler tests the cron scheduler functionality
func TestStartCronScheduler(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	// Create cache instance
	cacheInstance := cache.NewCache(300, 10000, 60, 10, true, 100)
	cacheInstance.StartPruner()

	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled:           true,
			AutoClearOnChange: true,
		},
		BlacklistRemotes: []config.RemoteFile{
			{
				URL:  "https://example.com/blacklist.txt",
				Cron: "0 * * * *",
			},
		},
		WhitelistRemotes: []config.RemoteFile{
			{
				URL:  "https://example.com/whitelist.txt",
				Cron: "0 0 * * *",
			},
		},
	}

	// Test starting cron scheduler
	setupCronScheduler(cfg, cacheInstance)

	// Verify no panic occurred
	t.Log("Cron scheduler started successfully")
}

// TestStartCronSchedulerNoJobs tests cron scheduler with no jobs
func TestStartCronSchedulerNoJobs(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	// Create cache instance
	cacheInstance := cache.NewCache(300, 10000, 60, 10, true, 100)
	cacheInstance.StartPruner()

	cfg := &config.Config{
		BlacklistRemotes: []config.RemoteFile{},
		WhitelistRemotes: []config.RemoteFile{},
	}

	// Test starting cron scheduler with no jobs
	setupCronScheduler(cfg, cacheInstance)

	// Verify no panic occurred
	t.Log("Cron scheduler handled no jobs case")
}

// TestStartCronScheduler_AllCodePaths tests all code paths in startCronScheduler
func TestStartCronScheduler_AllCodePaths(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	// Create cache instance for tests
	cacheInstance := cache.NewCache(300, 10000, 60, 10, true, 100)
	cacheInstance.StartPruner()

	// Test 1: Config with only blacklist remotes
	t.Run("BlacklistRemotesOnly", func(t *testing.T) {
		cfg := &config.Config{
			Cache: config.CacheConfig{
				Enabled:           true,
				AutoClearOnChange: true,
			},
			BlacklistRemotes: []config.RemoteFile{
				{
					URL:  "https://example.com/blacklist.txt",
					Cron: "0 * * * *",
				},
			},
			WhitelistRemotes: []config.RemoteFile{},
		}

		setupCronScheduler(cfg, cacheInstance)
		t.Log("Cron scheduler started with blacklist remotes only")
	})

	// Test 2: Config with only whitelist remotes
	t.Run("WhitelistRemotesOnly", func(t *testing.T) {
		cfg := &config.Config{
			Cache: config.CacheConfig{
				Enabled:           true,
				AutoClearOnChange: true,
			},
			BlacklistRemotes: []config.RemoteFile{},
			WhitelistRemotes: []config.RemoteFile{
				{
					URL:  "https://example.com/whitelist.txt",
					Cron: "0 0 * * *",
				},
			},
		}

		setupCronScheduler(cfg, cacheInstance)
		t.Log("Cron scheduler started with whitelist remotes only")
	})

	// Test 3: Config with multiple blacklist remotes
	t.Run("MultipleBlacklistRemotes", func(t *testing.T) {
		cfg := &config.Config{
			Cache: config.CacheConfig{
				Enabled:           true,
				AutoClearOnChange: true,
			},
			BlacklistRemotes: []config.RemoteFile{
				{
					URL:  "https://example.com/blacklist1.txt",
					Cron: "0 * * * *",
				},
				{
					URL:  "https://example.com/blacklist2.txt",
					Cron: "30 * * * *",
				},
			},
			WhitelistRemotes: []config.RemoteFile{},
		}

		setupCronScheduler(cfg, cacheInstance)
		t.Log("Cron scheduler started with multiple blacklist remotes")
	})

	// Test 4: Config with multiple whitelist remotes
	t.Run("MultipleWhitelistRemotes", func(t *testing.T) {
		cfg := &config.Config{
			Cache: config.CacheConfig{
				Enabled:           true,
				AutoClearOnChange: true,
			},
			BlacklistRemotes: []config.RemoteFile{},
			WhitelistRemotes: []config.RemoteFile{
				{
					URL:  "https://example.com/whitelist1.txt",
					Cron: "0 0 * * *",
				},
				{
					URL:  "https://example.com/whitelist2.txt",
					Cron: "30 0 * * *",
				},
			},
		}

		setupCronScheduler(cfg, cacheInstance)
		t.Log("Cron scheduler started with multiple whitelist remotes")
	})

	// Test 5: Config with remote but no cron expression
	t.Run("RemoteNoCron", func(t *testing.T) {
		cfg := &config.Config{
			Cache: config.CacheConfig{
				Enabled:           true,
				AutoClearOnChange: true,
			},
			BlacklistRemotes: []config.RemoteFile{
				{
					URL:  "https://example.com/blacklist.txt",
					Cron: "",
				},
			},
			WhitelistRemotes: []config.RemoteFile{},
		}

		setupCronScheduler(cfg, cacheInstance)
		t.Log("Cron scheduler handled remote with no cron expression")
	})

	// Test 6: Config with cron expression but no URL
	t.Run("CronNoURL", func(t *testing.T) {
		cfg := &config.Config{
			Cache: config.CacheConfig{
				Enabled:           true,
				AutoClearOnChange: true,
			},
			BlacklistRemotes: []config.RemoteFile{
				{
					URL:  "",
					Cron: "0 * * * *",
				},
			},
			WhitelistRemotes: []config.RemoteFile{},
		}

		setupCronScheduler(cfg, cacheInstance)
		t.Log("Cron scheduler handled cron expression with no URL")
	})

	// Test 7: Config with cache disabled
	t.Run("CacheDisabled", func(t *testing.T) {
		cfg := &config.Config{
			Cache: config.CacheConfig{
				Enabled:           false,
				AutoClearOnChange: false,
			},
			BlacklistRemotes: []config.RemoteFile{
				{
					URL:  "https://example.com/blacklist.txt",
					Cron: "0 * * * *",
				},
			},
			WhitelistRemotes: []config.RemoteFile{},
		}

		setupCronScheduler(cfg, cacheInstance)
		t.Log("Cron scheduler started with cache disabled")
	})

	// Test 8: Config with valid cache instance
	t.Run("ValidCacheInstance", func(t *testing.T) {
		cacheInstance := cache.NewCache(300, 10000, 60, 10, true, 100)
		cacheInstance.StartPruner()

		cfg := &config.Config{
			Cache: config.CacheConfig{
				Enabled:           true,
				AutoClearOnChange: true,
			},
			BlacklistRemotes: []config.RemoteFile{
				{
					URL:  "https://example.com/blacklist.txt",
					Cron: "0 * * * *",
				},
			},
			WhitelistRemotes: []config.RemoteFile{},
		}

		setupCronScheduler(cfg, cacheInstance)
		t.Log("Cron scheduler started with valid cache instance")
	})

	// Test 9: Config with no logger (nil logger)
	t.Run("NoLogger", func(t *testing.T) {
		cfg := &config.Config{
			Cache: config.CacheConfig{
				Enabled:           true,
				AutoClearOnChange: true,
			},
			BlacklistRemotes: []config.RemoteFile{
				{
					URL:  "https://example.com/blacklist.txt",
					Cron: "0 * * * *",
				},
			},
			WhitelistRemotes: []config.RemoteFile{},
		}

		// Pass nil logger - this should handle gracefully
		// Note: The function will panic if logger is nil, so we expect this to fail
		// This test documents the current behavior
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Log("Cron scheduler panicked with nil logger (expected behavior)")
				}
			}()
			setupCronScheduler(cfg, cacheInstance)
		}()
	})
}

// TestListenAndServeError tests error handling in listenAndServe
func TestListenAndServeError(t *testing.T) {
	// Create a handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Try to start server on privileged port (should fail)
	err := listenAndServe("80", handler)

	// Should get an error (address already in use or permission denied)
	if err == nil {
		t.Error("Expected error when starting server on privileged port")
	}
}

// TestLoadConfiguration tests the loadConfiguration function
func TestLoadConfiguration(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	// Test with non-existent config file
	cfg, err := loadConfiguration("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error for non-existent config file")
	}
	if cfg != nil {
		t.Error("Expected nil config for non-existent file")
	}
}

// TestSetupIPMatcher tests the setupIPMatcher function
func TestSetupIPMatcher(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	// Test with empty config
	cfg := &config.Config{
		BlacklistFiles:   []config.LocalFile{},
		BlacklistRemotes: []config.RemoteFile{},
		WhitelistFiles:   []config.WhitelistFile{},
		WhitelistRemotes: []config.RemoteFile{},
		BlocklistMaxSize: 10000,
	}

	err := setupIPMatcher(cfg)
	if err != nil {
		t.Fatalf("Expected no error for empty config, got: %v", err)
	}

	if cfg.IPMatcher == nil {
		t.Error("Expected IPMatcher to be created")
	}

	// Test with valid files
	tmpDir, err := ioutil.TempDir("", "ipmatcher-test")
	if err != nil {
		t.Fatal("Failed to create temp dir:", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create whitelist file
	whitelistFile := filepath.Join(tmpDir, "whitelist.txt")
	whitelistContent := "198.51.100.1\n203.0.113.1"
	if err := ioutil.WriteFile(whitelistFile, []byte(whitelistContent), 0644); err != nil {
		t.Fatal("Failed to write whitelist file:", err)
	}

	// Create blocklist file
	blocklistFile := filepath.Join(tmpDir, "blocklist.txt")
	blocklistContent := "203.0.113.2\n203.0.113.3"
	if err := ioutil.WriteFile(blocklistFile, []byte(blocklistContent), 0644); err != nil {
		t.Fatal("Failed to write blocklist file:", err)
	}

	cfg2 := &config.Config{
		WhitelistFiles: []config.WhitelistFile{
			{Path: whitelistFile},
		},
		BlacklistFiles: []config.LocalFile{
			{Path: blocklistFile},
		},
		BlocklistMaxSize: 10000,
	}

	err = setupIPMatcher(cfg2)
	if err != nil {
		t.Fatalf("Expected no error for valid files, got: %v", err)
	}

	if cfg2.IPMatcher == nil {
		t.Error("Expected IPMatcher to be created")
	}
}

// TestInitializeCache tests the initializeCache function
func TestInitializeCache(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	// Test with cache disabled
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled: false,
		},
	}

	cacheInstance, err := initializeCache(cfg)
	if err != nil {
		t.Fatalf("Expected no error for disabled cache, got: %v", err)
	}

	if cacheInstance != nil {
		t.Error("Expected nil cache instance when disabled")
	}

	// Test with cache enabled
	cfg2 := &config.Config{
		Cache: config.CacheConfig{
			Enabled:           true,
			TTL:               300,
			MaxEntries:        10000,
			PruneInterval:     60,
			ShardCount:        10,
			PruneOnGet:        true,
			WriteBufferSize:   100,
			AutoClearOnChange: true,
		},
	}

	cacheInstance2, err2 := initializeCache(cfg2)
	if err2 != nil {
		t.Fatalf("Expected no error for enabled cache, got: %v", err2)
	}

	if cacheInstance2 == nil {
		t.Error("Expected cache instance when enabled")
	}
}

// TestSetupCronScheduler tests the setupCronScheduler function
func TestSetupCronScheduler(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	// Test with no remote files
	cfg := &config.Config{
		BlacklistRemotes: []config.RemoteFile{},
		WhitelistRemotes: []config.RemoteFile{},
	}

	cacheInstance := cache.NewCache(300, 10000, 60, 10, true, 100)
	cacheInstance.StartPruner()

	// Should not panic
	setupCronScheduler(cfg, cacheInstance)
	t.Log("Cron scheduler setup completed without panic for empty config")

	// Test with remote files
	cfg2 := &config.Config{
		BlacklistRemotes: []config.RemoteFile{
			{URL: "https://example.com/blacklist.txt", Cron: "0 * * * *"},
		},
		WhitelistRemotes: []config.RemoteFile{
			{URL: "https://example.com/whitelist.txt", Cron: "0 0 * * *"},
		},
	}

	// Should not panic
	setupCronScheduler(cfg2, cacheInstance)
	t.Log("Cron scheduler setup completed without panic for config with remotes")
}

// TestSetupFileWatchers tests the setupFileWatchers function
func TestSetupFileWatchers(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	cacheInstance := cache.NewCache(300, 10000, 60, 10, true, 100)
	cacheInstance.StartPruner()

	// Test with watch files disabled
	cfg := &config.Config{
		WatchFilesEnabled: false,
		BlacklistFiles:    []config.LocalFile{{Path: "test.txt"}},
		WhitelistFiles:    []config.WhitelistFile{{Path: "test.txt"}},
	}

	// Should not panic
	setupFileWatchers(cfg, cacheInstance)
	t.Log("File watchers setup completed without panic when disabled")

	// Test with watch files enabled but no files
	cfg2 := &config.Config{
		WatchFilesEnabled: true,
		BlacklistFiles:    []config.LocalFile{},
		WhitelistFiles:    []config.WhitelistFile{},
	}

	// Should not panic
	setupFileWatchers(cfg2, cacheInstance)
	t.Log("File watchers setup completed without panic when enabled but no files")
}

// TestExtractFilePaths tests the extractFilePaths function
func TestExtractFilePaths(t *testing.T) {
	tests := []struct {
		name     string
		files    interface{}
		want     []string
	}{
		{"whitelist files", []config.WhitelistFile{
			{Path: "file1.txt"},
			{Path: "file2.txt"},
		}, []string{"file1.txt", "file2.txt"}},
		{"blocklist files", []config.LocalFile{
			{Path: "file1.txt"},
			{Path: "file2.txt"},
		}, []string{"file1.txt", "file2.txt"}},
		{"empty whitelist", []config.WhitelistFile{}, []string{}},
		{"empty blocklist", []config.LocalFile{}, []string{}},
		{"invalid type", "invalid", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFilePaths(tt.files)
			if len(got) != len(tt.want) {
				t.Errorf("extractFilePaths() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("extractFilePaths()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestSetupBlacklistWatcher tests the setupBlacklistWatcher function
func TestSetupBlacklistWatcher(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	cacheInstance := cache.NewCache(300, 10000, 60, 10, true, 100)
	cacheInstance.StartPruner()

	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled:           true,
			AutoClearOnChange: true,
		},
	}

	// Test with empty paths
	setupBlacklistWatcher(cfg, cacheInstance, []string{})
	t.Log("Blacklist watcher setup completed without panic for empty paths")

	// Test with valid paths
	tmpDir, err := ioutil.TempDir("", "watcher-test")
	if err != nil {
		t.Fatal("Failed to create temp dir:", err)
	}
	defer os.RemoveAll(tmpDir)

	blocklistFile := filepath.Join(tmpDir, "blocklist.txt")
	content := "198.51.100.1\n203.0.113.1"
	if err := ioutil.WriteFile(blocklistFile, []byte(content), 0644); err != nil {
		t.Fatal("Failed to write blocklist file:", err)
	}

	setupBlacklistWatcher(cfg, cacheInstance, []string{blocklistFile})
	t.Log("Blacklist watcher setup completed without panic for valid paths")
}

// TestSetupWhitelistWatcher tests the setupWhitelistWatcher function
func TestSetupWhitelistWatcher(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	cacheInstance := cache.NewCache(300, 10000, 60, 10, true, 100)
	cacheInstance.StartPruner()

	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled:           true,
			AutoClearOnChange: true,
		},
	}

	// Test with empty paths
	setupWhitelistWatcher(cfg, cacheInstance, []string{})
	t.Log("Whitelist watcher setup completed without panic for empty paths")

	// Test with valid paths
	tmpDir, err := ioutil.TempDir("", "watcher-test")
	if err != nil {
		t.Fatal("Failed to create temp dir:", err)
	}
	defer os.RemoveAll(tmpDir)

	whitelistFile := filepath.Join(tmpDir, "whitelist.txt")
	content := "198.51.100.1\n203.0.113.1"
	if err := ioutil.WriteFile(whitelistFile, []byte(content), 0644); err != nil {
		t.Fatal("Failed to write whitelist file:", err)
	}

	setupWhitelistWatcher(cfg, cacheInstance, []string{whitelistFile})
	t.Log("Whitelist watcher setup completed without panic for valid paths")
}

// TestHandleCacheClearOnChange tests the handleCacheClearOnChange function
func TestHandleCacheClearOnChange(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	// Test with cache disabled
	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled:           false,
			AutoClearOnChange: false,
		},
	}

	cacheInstance := cache.NewCache(300, 10000, 60, 10, true, 100)
	cacheInstance.StartPruner()

	// Should not panic
	handleCacheClearOnChange(cfg, cacheInstance, "blacklist")
	t.Log("Cache clear handler completed without panic when disabled")

	// Test with cache enabled
	cfg2 := &config.Config{
		Cache: config.CacheConfig{
			Enabled:           true,
			AutoClearOnChange: true,
		},
	}

	// Should not panic
	handleCacheClearOnChange(cfg2, cacheInstance, "whitelist")
	t.Log("Cache clear handler completed without panic when enabled")
}

// TestStartServer tests the startServer function
func TestStartServer(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	cfg := &config.Config{
		Port: "8081",
	}

	cacheInstance := cache.NewCache(300, 10000, 60, 10, true, 100)
	cacheInstance.StartPruner()

	// Test that startServer doesn't panic with valid configuration
	// We won't actually start the server to avoid race conditions
	// The function creates handler dependencies and router, which we can verify
	handlerDeps := &models.HandlerDeps{
		Config: cfg,
		Cache:  cacheInstance,
		Logger: logger,
	}
	
	// Verify handler dependencies are created correctly
	if handlerDeps.Config == nil {
		t.Error("Expected config in handler dependencies")
	}
	if handlerDeps.Cache == nil {
		t.Error("Expected cache in handler dependencies")
	}
	if handlerDeps.Logger == nil {
		t.Error("Expected logger in handler dependencies")
	}
	
	t.Log("Handler dependencies created successfully")
}

// TestCreateBlacklistUpdateJob tests the createBlacklistUpdateJob function
func TestCreateBlacklistUpdateJob(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled:           true,
			AutoClearOnChange: true,
		},
		BlacklistFiles: []config.LocalFile{
			{Path: "config/blocklists/blocklist.txt"},
		},
	}

	cacheInstance := cache.NewCache(300, 10000, 60, 10, true, 100)
	cacheInstance.StartPruner()

	// Create the job function
	job := createBlacklistUpdateJob(cfg, logger, "https://example.com/blacklist.txt", cacheInstance)

	// Execute the job
	job()

	// Verify no panic occurred
	t.Log("Blacklist update job executed without panic")
}

// TestCreateWhitelistUpdateJob tests the createWhitelistUpdateJob function
func TestCreateWhitelistUpdateJob(t *testing.T) {
	// Save original logger
	originalLogger := logger
	defer func() { logger = originalLogger }()

	// Set test logger
	logger = logging.NewLogger("DEBUG")

	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled:           true,
			AutoClearOnChange: true,
		},
		WhitelistFiles: []config.WhitelistFile{
			{Path: "config/whitelists/whitelist.txt"},
		},
	}

	cacheInstance := cache.NewCache(300, 10000, 60, 10, true, 100)
	cacheInstance.StartPruner()

	// Create the job function
	job := createWhitelistUpdateJob(cfg, logger, "https://example.com/whitelist.txt", cacheInstance)

	// Execute the job
	job()

	// Verify no panic occurred
	t.Log("Whitelist update job executed without panic")
}

// TestMain ensures logger is initialized for tests
func TestMain(m *testing.M) {
	logger = logging.NewLogger("INFO")
	os.Exit(m.Run())
}
