// Package models provides data structures and types used throughout the IPGate application.
// It includes request/response models, error page data, and dependency injection structures.
//
// Key components:
// - ErrorPageData: Data structure for rendering error pages
// - Response models: Structured responses for API endpoints
// - HandlerDeps: Dependency injection container for handlers
// - ResponseWriter: Wrapper for tracking response details
package models

import (
	"github.com/dombyte/ipgate/internal/cache"
	"github.com/dombyte/ipgate/internal/config"
	"github.com/dombyte/ipgate/internal/logging"
	"net/http"
)

// ErrorPageData contains all necessary data for rendering error pages
// when an IP is blocked. This structure is used to populate both HTML
// and JSON error responses.
type ErrorPageData struct {
	RequestID   string            // Unique request identifier for tracing
	ClientIP    string            // The IP address that was blocked
	Host        string            // Host header from the request
	URI         string            // Request URI
	Method      string            // HTTP method used
	Proto       string            // Protocol version
	Headers     map[string]string // Request headers (first value only)
	StatusCode  int               // HTTP status code to return
	BlockReason string            // Reason for blocking (e.g., "matched CIDR", "whitelisted")
}

// TestIPResponse represents the response for the test IP debug endpoint.
// It provides information about whether a specific IP would be blocked
// or allowed.
type TestIPResponse struct {
	IP      string `json:"ip"`      // The IP address tested
	Blocked bool   `json:"blocked"` // Whether the IP is blocked (true) or allowed (false)
	Reason  string `json:"reason"`  // Reason for the decision (e.g., "matched CIDR", "whitelisted")
	Status  string `json:"status"`  // Cache status: "ALLOW", "DENY", or "MISS"
}

// HealthResponse represents the response for the health check endpoint.
// It provides a simple status indicator to verify the service is running.
type HealthResponse struct {
	Status string `json:"status"` // Service status: "ok" if healthy
}

// CacheEntry represents a single cache entry containing the status
// and timestamp of an IP blocking decision.
// Status can be "ALLOW" (whitelisted), "DENY" (blocked), or "MISS" (not found).
// Timestamp indicates when the entry was added or last updated.
type CacheEntry struct {
	Status    string // Cache status: ALLOW, DENY, or MISS
	Timestamp int64  // Unix timestamp when the entry was created or updated
}

// CacheDumpResponse represents the response for the cache dump debug endpoint.
// It provides a complete snapshot of all current cache entries for debugging
// and monitoring purposes.
type CacheDumpResponse struct {
	Entries map[string]CacheEntry `json:"entries"` // Map of cache keys to CacheEntry values
}

// ResponseWriter wraps http.ResponseWriter to track status code and
// bytes written. This is used for tracking response details.
type ResponseWriter struct {
	http.ResponseWriter     // Embedded ResponseWriter for delegation
	StatusCode          int // HTTP status code that was written
	BytesWritten        int // Total bytes written in the response
}

// WriteHeader tracks the status code while delegating to the underlying
// ResponseWriter.
//
// Parameters:
//
//	code - HTTP status code to write
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write tracks the number of bytes written while delegating to the underlying
// ResponseWriter.
//
// Parameters:
//
//	b - Bytes to write
//
// Returns:
//
//	int - Number of bytes written
//	error - Any error that occurred during writing
func (rw *ResponseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.BytesWritten += size
	return size, err
}

// HandlerDeps contains all dependencies required by HTTP handlers.
// This structure is used for dependency injection, making handlers
// easier to test and more modular.
//
// Key dependencies:
// - Config: Application configuration
// - Cache: IP blocking cache for performance optimization
// - Additional dependencies can be added as needed
//
// Example usage:
//
//	deps := &models.HandlerDeps{
//	    Config: config,
//	    Cache:  cache,
//	}
//	handler := handler.BouncerHandler(deps)
type HandlerDeps struct {
	Config *config.Config  // Application configuration
	Cache  *cache.Cache    // IP blocking cache
	Logger *logging.Logger // Logger for structured logging
}
