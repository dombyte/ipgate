// Package handler contains HTTP request handlers for the IPGate service.
// This package has been restructured to separate main handlers from debug handlers.
// For debug handlers, see debug_handlers.go.
// For middleware, see the middleware package.
package handler

// This file is kept for backward compatibility and will be removed in future versions.
// All handler functionality has been moved to:
// - handlers.go: Main public API handlers (BouncerHandler, HealthHandler)
// - debug_handlers.go: Debug and administrative endpoints
// - middleware package: HTTP middleware functions
