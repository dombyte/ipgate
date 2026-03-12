// Package routes provides route definitions and router configuration for the IPGate application.
// It uses the chi router for flexible route composition and middleware support.
//
// Key features:
// - Centralized route management
// - Middleware composition
// - Conditional route registration based on configuration
// - Clean separation of concerns
package routes

import (
	"github.com/dombyte/ipgate/internal/handler"
	"github.com/dombyte/ipgate/internal/middleware"
	"github.com/dombyte/ipgate/internal/models"
	"github.com/go-chi/chi/v5"
	"net/http"
)

// NewRouter creates a new HTTP router with all routes and middleware.
// This function sets up the complete routing configuration for the application,
// including public endpoints, debug endpoints (when enabled), and global middleware.
//
// The router:
// - Creates a new chi router instance
// - Applies global middleware (request logging)
// - Registers public routes (health, bouncer)
// - Conditionally registers debug routes based on configuration
// - Returns the fully configured router
//
// Parameters:
//
//	deps - Handler dependencies including config and cache
//
// Returns:
//
//	http.Handler - Fully configured HTTP router
//
// Example:
//
//	deps := &models.HandlerDeps{Config: config, Cache: cache}
//	router := NewRouter(deps)
//	http.ListenAndServe(":8080", router)
func NewRouter(deps *models.HandlerDeps) http.Handler {
	mux := chi.NewRouter()

	// Apply global middleware
	mux.Use(middleware.RequestLoggingMiddleware(deps))

	// Public routes
	mux.Handle("/health", handler.HealthHandler())
	mux.Handle("/bouncer", handler.BouncerHandler(deps))

	// Debug routes (conditionally enabled based on config)
	if deps.Config.DebugEndpoint {
		mux.Handle("/debug/clear-cache", handler.ClearCacheHandler(deps))
		mux.Handle("/debug/cache-dump", handler.CacheDumpHandler(deps))
		mux.Handle("/debug/config", handler.ConfigHandler(deps))
	}

	return mux
}
