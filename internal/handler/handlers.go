// Package handler provides HTTP request handlers for the IPGate application.
// It includes the main bouncer handler for IP blocking logic and health
// check endpoints. Handlers use dependency injection via the HandlerDeps
// structure for easy testing and modularity.
//
// Key features:
// - Bouncer handler with cache integration
// - Health check endpoint
// - HTML and JSON error response support
// - Thread-safe dependency injection
package handler

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/dombyte/ipgate/internal/blocklist"
	"github.com/dombyte/ipgate/internal/ipmatcher"
	"github.com/dombyte/ipgate/internal/models"
	"github.com/google/uuid"
)

// BouncerHandler handles IP blocking logic for the /bouncer endpoint.
// This is the main handler that processes incoming requests and determines
// whether to allow or block access based on IP blocklists and whitelists.
//
// The handler:
// - Retrieves the client IP from the configured header
// - Checks cache first for performance (if enabled)
// - Falls back to blocklist/whitelist checking on cache miss
// - Returns HTML or JSON error responses based on configuration
// - Supports both IPv4 and IPv6 addresses
//
// Parameters:
//
//	deps - Handler dependencies including config and cache
//
// Returns:
//
//	http.Handler - HTTP handler function
//
// Example:
//
//	deps := &models.HandlerDeps{Config: config, Cache: cache}
//	router.Handle("/bouncer", BouncerHandler(deps))
func BouncerHandler(deps *models.HandlerDeps) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Extract and validate client IP
		clientIP, err := extractClientIP(deps, w, r)
		if err != nil {
			return
		}

		// Check cache status
		cacheStatus, err := getCacheStatus(deps, clientIP)
		if err != nil {
			return
		}

		// Handle cached DENY
		if cacheStatus == "DENY" {
			handleCachedDeny(deps, w, r, clientIP)
			return
		}

		// Handle cached ALLOW
		if cacheStatus == "ALLOW" {
			handleCachedAllow(deps, w, clientIP)
			return
		}

		// Cache miss - check actual blocklists
		err = handleCacheMiss(deps, w, r, clientIP, startTime)
		if err != nil {
			return
		}
	})
}

// extractClientIP extracts and validates the client IP from request headers
func extractClientIP(deps *models.HandlerDeps, w http.ResponseWriter, r *http.Request) (string, error) {
	clientIP := r.Header.Get(deps.Config.Headers["client_ip_header"])
	if clientIP == "" {
		deps.Logger.Warn("No client IP found in header %s", "header", deps.Config.Headers["client_ip_header"])
		http.Error(w, "Client IP not found", http.StatusBadRequest)
		return "", fmt.Errorf("client IP not found")
	}
	return clientIP, nil
}

// getCacheStatus checks the cache for the IP status
func getCacheStatus(deps *models.HandlerDeps, clientIP string) (string, error) {
	// New simplified logic: Check cache first (if enabled)
	// IMPORTANT: Cache hits do NOT reset TTL. This prevents stale entries from staying
	// in cache forever when IPs are requested frequently.
	cacheStatus := "MISS"
	if deps.Cache != nil {
		cacheStatus = deps.Cache.GetStatus(clientIP)
	}
	return cacheStatus, nil
}

// handleCachedDeny handles the case when IP is cached as DENY
func handleCachedDeny(deps *models.HandlerDeps, w http.ResponseWriter, r *http.Request, clientIP string) {
	deps.Logger.Info("IP %s is cached as DENY", "ip", clientIP)

	data := buildErrorPageData(deps, r, clientIP, "cached")

	if deps.Config.ErrorFormat == "html" {
		renderHTMLResponse(deps, w, data, deps.Config.StatusDenied)
	} else {
		renderJSONResponse(w, deps.Config.StatusDenied, map[string]string{"error": "access denied", "reason": "cached"})
	}
}

// handleCachedAllow handles the case when IP is cached as ALLOW
func handleCachedAllow(deps *models.HandlerDeps, w http.ResponseWriter, clientIP string) {
	// IP is cached as whitelisted - allow access without checking blocklists
	deps.Logger.Info("IP %s is cached as ALLOW", "ip", clientIP)
	w.WriteHeader(deps.Config.StatusAllowed)
}

// handleCacheMiss handles the case when cache misses and we need to check actual blocklists
func handleCacheMiss(deps *models.HandlerDeps, w http.ResponseWriter, r *http.Request, clientIP string, startTime time.Time) error {
	// Get the IPMatcher from the config
	matcher, ok := deps.Config.IPMatcher.(*ipmatcher.IPMatcher)
	if !ok {
		deps.Logger.Error("Error: IPMatcher not initialized")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return fmt.Errorf("IPMatcher not initialized")
	}

	isBlocked, reason, err := blocklist.IsIPBlocked(clientIP, matcher)
	if err != nil {
		deps.Logger.Error("Error checking IP %s: %v", "ip", clientIP, "error", err)
		http.Error(w, "Invalid IP address", http.StatusBadRequest)
		return fmt.Errorf("error checking IP: %w", err)
	}

	if !isBlocked {
		// IP is Allowed - cache as ALLOW and allow access (if cache enabled)
		if deps.Cache != nil {
			deps.Cache.Add(clientIP, "ALLOW")
		}
		deps.Logger.Info("IP %s is allowed", "ip", clientIP)
		w.WriteHeader(deps.Config.StatusAllowed)
		return nil
	}

	// IP is not whitelisted - cache as DENY and block access (if cache enabled)
	if deps.Cache != nil {
		deps.Cache.Add(clientIP, "DENY")
	}
	deps.Logger.Info("IP %s is blocked: %s", "ip", clientIP, "reason", reason)

	data := buildErrorPageData(deps, r, clientIP, reason)

	if deps.Config.ErrorFormat == "html" {
		renderHTMLResponse(deps, w, data, deps.Config.StatusDenied)
	} else {
		renderJSONResponse(w, deps.Config.StatusDenied, map[string]string{"error": "access denied", "reason": reason})
	}

	// Record request duration and response size
	duration := time.Since(startTime)

	// Get response size (approximate)
	if rw, ok := w.(*models.ResponseWriter); ok {
		deps.Logger.Debug("Request completed in %v, response size: %d bytes", "duration", duration, "status", rw.StatusCode)
	}

	return nil
}

// buildErrorPageData builds the error page data structure
func buildErrorPageData(deps *models.HandlerDeps, r *http.Request, clientIP, reason string) models.ErrorPageData {
	data := models.ErrorPageData{
		RequestID:   uuid.New().String(),
		ClientIP:    clientIP,
		Host:        r.Header.Get(deps.Config.Headers["host_header"]),
		URI:         r.Header.Get(deps.Config.Headers["uri_header"]),
		Method:      r.Header.Get(deps.Config.Headers["method_header"]),
		Proto:       r.Header.Get(deps.Config.Headers["proto_header"]),
		Headers:     make(map[string]string),
		StatusCode:  deps.Config.StatusDenied,
		BlockReason: reason,
	}
	for key, values := range r.Header {
		if len(values) > 0 {
			data.Headers[key] = values[0]
		}
	}
	return data
}

// renderHTMLResponse renders an HTML error response
func renderHTMLResponse(deps *models.HandlerDeps, w http.ResponseWriter, data models.ErrorPageData, statusCode int) {
	tmpl, err := template.ParseFiles(deps.Config.ErrorPage)
	if err != nil {
		deps.Logger.Error("Failed to parse template: %v", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(statusCode)
	if err := tmpl.Execute(w, data); err != nil {
		deps.Logger.Error("Failed to execute template: %v", "error", err)
	}
}

// renderJSONResponse renders a JSON error response
func renderJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but don't return error to client since we already wrote header
		// This prevents "http: superfluous response.WriteHeader call" errors
	}
}

// HealthHandler returns health status for the /health endpoint.
// This is a simple endpoint that returns a JSON response indicating
// the service is running and healthy. It's commonly used by load
// balancers and monitoring systems to check service availability.
//
// Returns:
//
//	http.HandlerFunc - HTTP handler function that returns health status
//
// Example:
//
//	router.Handle("/health", HealthHandler())
//
// Response:
//
//	{"status": "ok"}
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			// Log error but continue - health check should not fail due to encoding issues
		}
	}
}
