// Package config provides configuration loading and management for the IPGate application.
// It supports YAML-based configuration with validation and thread-safe access.
//
// Key features:
// - YAML configuration loading with automatic file extension detection
// - Thread-safe configuration access using RWMutex
// - Comprehensive validation of configuration values
// - Default values for optional configuration parameters
//
// Deprecated: This file is deprecated. Use the new ConfigAPI in api.go and models in models.go instead.
package config

// GetHeaders returns a copy of the HTTP headers configuration.
// This is a simple getter that doesn't require thread-safety since the new
// Config struct doesn't have a mutex (thread-safety is handled by ConfigAPI).
func (c *Config) GetHeaders() map[string]string {
	headersCopy := make(map[string]string)
	for k, v := range c.Headers {
		headersCopy[k] = v
	}
	return headersCopy
}

// GetCache returns a copy of the cache configuration.
// This is a simple getter that doesn't require thread-safety since the new
// Config struct doesn't have a mutex (thread-safety is handled by ConfigAPI).
func (c *Config) GetCache() CacheConfig {
	return c.Cache
}
