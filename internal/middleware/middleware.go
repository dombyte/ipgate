// Package middleware provides HTTP middleware functions for the IPGate application.
// Middleware functions are used to wrap HTTP handlers with additional functionality
// such as logging, authentication, and rate limiting. They follow the standard
// Go middleware pattern of returning func(http.Handler) http.Handler.
//
// Key features:
// - Request logging
// - Authentication for debug endpoints
// - Rate limiting for API protection
// - Dependency injection for configuration and state
package middleware

import (
	"net/http"
	"time"

	"github.com/dombyte/ipgate/internal/models"
)

// RequestLoggingMiddleware logs HTTP requests with timing information.
// For ALLOW (200) responses, logging is done asynchronously to reduce latency,
// while error responses are logged synchronously to ensure errors are captured
// immediately.
//
// The middleware:
// - Wraps the ResponseWriter to track status code and response size
// - Measures request duration
// - Logs request method, path, remote address, status code, and duration
// - Uses async logging for performance on successful requests
// - Uses sync logging for error responses to preserve error handling behavior
//
// Parameters:
//
//	deps - Handler dependencies (currently unused but available for future extensions)
//
// Returns:
//
//	func(http.Handler) http.Handler - Middleware function that wraps handlers
//
// Example:
//
//	router.Use(RequestLoggingMiddleware(deps))
func RequestLoggingMiddleware(deps *models.HandlerDeps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &models.ResponseWriter{ResponseWriter: w}
			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			// Async logging for ALLOW responses (non-blocking) to improve performance
			// Sync logging for non-200 responses to preserve error handling behavior
			if deps != nil && deps.Logger != nil {
				if rw.StatusCode == http.StatusOK {
					go func() {
						deps.Logger.Info("Request handled", "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr, "status", rw.StatusCode, "duration", duration)
					}()
				} else {
					// Sync logging for error responses to ensure errors are logged immediately
					deps.Logger.Info("Request handled", "method", r.Method, "path", r.URL.Path, "remote_addr", r.RemoteAddr, "status", rw.StatusCode, "duration", duration)
				}
			}
		})
	}
}
