// Package config provides configuration loading and management for the IPGate application.
// This file contains the data models for application configuration.
package config

// Config represents the application configuration.
// All fields are populated from YAML config file via ConfigAPI.
type Config struct {
	// Server configuration
	Port        string `yaml:"port" validate:"required"`
	ErrorPage   string `yaml:"error_page"`
	ErrorFormat string `yaml:"error_format"`

	// HTTP status codes
	StatusAllowed int `yaml:"status_allowed"`
	StatusDenied  int `yaml:"status_denied"`

	// HTTP headers
	Headers map[string]string `yaml:"headers"`

	// Cache configuration
	Cache CacheConfig `yaml:"cache"`

	// File size limits
	BlocklistMaxSize int64 `yaml:"blocklist_max_size"`

	// Feature flags
	DebugEndpoint     bool `yaml:"debug_endpoint"`
	WatchFilesEnabled bool `yaml:"watch_files_enabled"`

	// Whitelist sources
	WhitelistFiles   []WhitelistFile `yaml:"whitelist_files"`
	WhitelistRemotes []RemoteFile    `yaml:"whitelist_remotes"`

	// Blacklist sources
	BlacklistFiles   []LocalFile  `yaml:"blocklist_files"`
	BlacklistRemotes []RemoteFile `yaml:"blocklist_remotes"`

	// Internal runtime state (not from config)
	IPMatcher  interface{} `yaml:"-"`
	Blocklists [][]string  `yaml:"-"`
	Whitelist  []string    `yaml:"-"`
}

// CacheConfig contains cache-specific settings
type CacheConfig struct {
	Enabled           bool `yaml:"enabled"`
	TTL               int  `yaml:"ttl"`
	MaxEntries        int  `yaml:"max_entries"`
	PruneInterval     int  `yaml:"prune_interval"`
	ShardCount        int  `yaml:"shard_count"`
	PruneOnGet        bool `yaml:"prune_on_get"`
	WriteBufferSize   int  `yaml:"write_buffer_size"`
	AutoClearOnChange bool `yaml:"auto_clear_on_change"`
}

// RemoteFile represents a remote config source
type RemoteFile struct {
	URL  string `yaml:"url"`
	Cron string `yaml:"cron"`
}

// LocalFile represents a local file source
type LocalFile struct {
	Path string `yaml:"path"`
}

// WhitelistFile represents a whitelist file source
type WhitelistFile struct {
	Path string `yaml:"path"`
}
