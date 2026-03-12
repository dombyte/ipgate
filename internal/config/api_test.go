// Package config provides configuration loading and management for the IPGate application.
// This file contains tests for the ConfigAPI.

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestConfigAPIBooleanValues tests that boolean configuration values are respected
// and not overridden by defaults when explicitly set to false
func TestConfigAPIBooleanValues(t *testing.T) {
	testCases := []struct {
		name                    string
		content                 string
		expectCacheEnabled      bool
		expectAutoClearOnChange bool
		expectPruneOnGet        bool
	}{
		{
			name: "Cache enabled false should be respected",
			content: `port: "9090"
cache:
  enabled: false
  ttl: 300
  auto_clear_on_change: true`,
			expectCacheEnabled:      false,
			expectAutoClearOnChange: true,
			expectPruneOnGet:        false, // default
		},
		{
			name: "Auto clear on change false should be respected",
			content: `port: "9090"
cache:
  enabled: true
  ttl: 300
  auto_clear_on_change: false`,
			expectCacheEnabled:      true,
			expectAutoClearOnChange: false,
			expectPruneOnGet:        false, // default
		},
		{
			name: "Prune on get true should be respected",
			content: `port: "9090"
cache:
  enabled: true
  ttl: 300
  prune_on_get: true
  auto_clear_on_change: true`,
			expectCacheEnabled:      true,
			expectAutoClearOnChange: true,
			expectPruneOnGet:        true,
		},
		{
			name: "All boolean values explicitly set",
			content: `port: "9090"
cache:
  enabled: false
  ttl: 300
  auto_clear_on_change: false
  prune_on_get: true`,
			expectCacheEnabled:      false,
			expectAutoClearOnChange: false,
			expectPruneOnGet:        true,
		},
		{
			name: "No cache section - should apply defaults",
			content: `port: "9090"
cache:
  ttl: 300
  auto_clear_on_change: true`,
			expectCacheEnabled:      true,  // default
			expectAutoClearOnChange: true,  // default
			expectPruneOnGet:        false, // default
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "config.yaml")

			err := os.WriteFile(tmpFile, []byte(tc.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write config file: %v", err)
			}

			configAPI := NewConfigAPI()
			err = configAPI.LoadConfig(tmpFile)
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			config, err := configAPI.GetConfig()
			if err != nil {
				t.Fatalf("Failed to get config: %v", err)
			}

			if config.Cache.Enabled != tc.expectCacheEnabled {
				t.Errorf("Expected cache.enabled=%v, got %v", tc.expectCacheEnabled, config.Cache.Enabled)
			}

			if config.Cache.AutoClearOnChange != tc.expectAutoClearOnChange {
				t.Errorf("Expected cache.auto_clear_on_change=%v, got %v", tc.expectAutoClearOnChange, config.Cache.AutoClearOnChange)
			}

			if config.Cache.PruneOnGet != tc.expectPruneOnGet {
				t.Errorf("Expected cache.prune_on_get=%v, got %v", tc.expectPruneOnGet, config.Cache.PruneOnGet)
			}
		})
	}
}

// TestConfigAPIExistsCheck tests that the Exists method works correctly for boolean values
func TestConfigAPIExistsCheck(t *testing.T) {
	configAPI := NewConfigAPI()

	// Test with a config that has cache.enabled explicitly set to false
	configContent := `port: "9090"
cache:
  enabled: false
  ttl: 300`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	err = configAPI.LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check that the key exists
	if !configAPI.k.Exists("cache.enabled") {
		t.Error("Expected cache.enabled to exist in config")
	}

	// Check that the value is false
	if configAPI.k.Bool("cache.enabled") != false {
		t.Error("Expected cache.enabled to be false")
	}
}

// TestLoadFile tests the loadFile method
func TestLoadFile(t *testing.T) {
	configAPI := NewConfigAPI()

	// Create a test config file
	configContent := `port: "9090"
cache:
  enabled: true
  ttl: 300`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test loading with full path
	err = configAPI.loadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	// Verify the config was loaded
	if !configAPI.k.Exists("port") {
		t.Error("Expected port to exist in config")
	}

	if configAPI.k.String("port") != "9090" {
		t.Errorf("Expected port to be 9090, got %s", configAPI.k.String("port"))
	}
}

// TestLoadFileAutoDetect tests the loadFile method with auto-detection
func TestLoadFileAutoDetect(t *testing.T) {
	configAPI := NewConfigAPI()

	// Create a test config file without extension
	configContent := `port: "9090"
cache:
  enabled: true
  ttl: 300`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config")

	err := os.WriteFile(tmpFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test loading without extension (should auto-detect)
	err = configAPI.loadFile(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	// Verify the config was loaded
	if !configAPI.k.Exists("port") {
		t.Error("Expected port to exist in config")
	}

	if configAPI.k.String("port") != "9090" {
		t.Errorf("Expected port to be 9090, got %s", configAPI.k.String("port"))
	}
}
