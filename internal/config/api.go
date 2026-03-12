// Package config provides configuration loading and management for the IPGate application.
// This file contains the ConfigAPI implementation using Koanf.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// ConfigAPI provides thread-safe access to application configuration
type ConfigAPI struct {
	k  *koanf.Koanf
	mu sync.RWMutex
}

// NewConfigAPI creates a new ConfigAPI instance
func NewConfigAPI() *ConfigAPI {
	return &ConfigAPI{
		k: koanf.New("."),
	}
}

// LoadConfig loads configuration with default lookup order:
// 1. [WORK_DIR]/ipgate.yml or .yaml
// 2. [WORK_DIR]/config.yml or .yaml
// 3. [WORK_DIR]/config/ipgate.yml or .yaml
// 4. [WORK_DIR]/config/config.yml or .yaml
func (c *ConfigAPI) LoadConfig(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If path provided, use it directly
	if path != "" {
		return c.loadFile(path)
	}

	// Default lookup order
	workDir := "."
	paths := []string{
		filepath.Join(workDir, "ipgate"),
		filepath.Join(workDir, "config"),
		filepath.Join(workDir, "config", "ipgate"),
		filepath.Join(workDir, "config", "config"),
	}

	for _, base := range paths {
		for _, ext := range []string{".yaml", ".yml"} {
			fullPath := base + ext
			if _, err := os.Stat(fullPath); err == nil {
				return c.loadFile(fullPath)
			}
		}
	}

	return fmt.Errorf("no config file found in default locations")
}

func (c *ConfigAPI) loadFile(path string) error {
	// Auto-detect .yaml/.yml if needed
	if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
		for _, ext := range []string{".yaml", ".yml"} {
			if _, err := os.Stat(path + ext); err == nil {
				path += ext
				break
			}
		}
	}

	return c.k.Load(file.Provider(path), yaml.Parser())
}

// ConfigWithMetadata contains configuration with metadata about sources
type ConfigWithMetadata struct {
	Config          *Config
	DefaultsApplied map[string]bool
	FileContents    map[string]interface{}
}

// GetConfig returns a fully populated Config struct with defaults applied
func (c *ConfigAPI) GetConfig() (*Config, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cfg := &Config{}
	// Use yaml tags instead of koanf tags since our struct uses yaml tags
	if err := c.k.UnmarshalWithConf("", cfg, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return nil, err
	}

	// Boolean and file configurations are now properly unmarshaled using yaml tags

	c.applyDefaults(cfg)
	return cfg, nil
}

func (c *ConfigAPI) applyDefaults(cfg *Config) {
	// Apply error page and format defaults
	applyErrorPageDefaults(cfg)

	// Apply status code defaults
	applyStatusCodeDefaults(cfg)

	// Apply blocklist defaults
	applyBlocklistDefaults(cfg)

	// Apply cache defaults
	applyCacheDefaults(c, cfg)
}

// applyErrorPageDefaults applies defaults for error page configuration
func applyErrorPageDefaults(cfg *Config) {
	if cfg.ErrorPage == "" {
		cfg.ErrorPage = "/app/templates/error.html"
	}
	if cfg.ErrorFormat == "" {
		cfg.ErrorFormat = "html"
	}
}

// applyStatusCodeDefaults applies defaults for HTTP status codes
func applyStatusCodeDefaults(cfg *Config) {
	if cfg.StatusAllowed == 0 {
		cfg.StatusAllowed = 200
	}
	if cfg.StatusDenied == 0 {
		cfg.StatusDenied = 403
	}
}

// applyBlocklistDefaults applies defaults for blocklist configuration
func applyBlocklistDefaults(cfg *Config) {
	if cfg.BlocklistMaxSize == 0 {
		cfg.BlocklistMaxSize = 10485760 // 10MB
	}
}

// applyCacheDefaults applies defaults for cache configuration
func applyCacheDefaults(c *ConfigAPI, cfg *Config) {
	// Only apply cache.enabled default if it wasn't explicitly set in config
	if !c.k.Exists("cache.enabled") {
		cfg.Cache.Enabled = false
	}
	if cfg.Cache.TTL == 0 {
		cfg.Cache.TTL = 300
	}
	if cfg.Cache.MaxEntries == 0 {
		cfg.Cache.MaxEntries = 100000
	}
	if cfg.Cache.PruneInterval == 0 {
		cfg.Cache.PruneInterval = 60
	}
	if cfg.Cache.ShardCount == 0 {
		cfg.Cache.ShardCount = 64
	}
	if cfg.Cache.WriteBufferSize == 0 {
		cfg.Cache.WriteBufferSize = 0
	}
	// Only apply cache.prune_on_get default if it wasn't explicitly set in config
	if !c.k.Exists("cache.prune_on_get") {
		cfg.Cache.PruneOnGet = false
	}
}
