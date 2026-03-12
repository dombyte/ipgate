package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dombyte/ipgate/internal/cache"
	"github.com/dombyte/ipgate/internal/config"
	"github.com/dombyte/ipgate/internal/logging"
	"github.com/dombyte/ipgate/internal/models"
)

// TestDebugRoutesEnabled tests that debug routes are registered when debug_enabled is true
func TestDebugRoutesEnabled(t *testing.T) {
	// Create test config with debug_endpoint: true
	testConfig := &config.Config{
		Port:             "8080",
		StatusAllowed:    200,
		StatusDenied:     403,
		ErrorFormat:      "json",
		ErrorPage:        "templates/error.html",
		BlocklistMaxSize: 1024 * 1024,
		DebugEndpoint:    true,
		Headers: map[string]string{
			"client_ip_header": "X-Forwarded-For",
		},
	}

	// Create test cache
	testCache := cache.NewCache(300, 10000, 60, 64, false, 0)

	// Create handler dependencies
	deps := &models.HandlerDeps{
		Config: testConfig,
		Cache:  testCache,
		Logger: logging.NewLogger("INFO"),
	}

	// Create router
	router := NewRouter(deps)

	// Test that debug endpoints are accessible
	testCases := []struct {
		path       string
		statusCode int
	}{
		{"/debug/clear-cache", http.StatusOK},
		{"/debug/cache-dump", http.StatusOK},
		{"/debug/config", http.StatusOK},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest("GET", tc.path, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != tc.statusCode {
			t.Errorf("Expected status %d for %s, got %d", tc.statusCode, tc.path, rr.Code)
		}
	}
}

// TestDebugRoutesDisabled tests that debug routes are NOT registered when debug_enabled is false
func TestDebugRoutesDisabled(t *testing.T) {
	// Create test config with debug_endpoint: false
	testConfig := &config.Config{
		Port:             "8080",
		StatusAllowed:    200,
		StatusDenied:     403,
		ErrorFormat:      "json",
		ErrorPage:        "templates/error.html",
		BlocklistMaxSize: 1024 * 1024,
		DebugEndpoint:    false,
		Headers: map[string]string{
			"client_ip_header": "X-Forwarded-For",
		},
	}

	// Create test cache
	testCache := cache.NewCache(300, 10000, 60, 64, false, 0)

	// Create handler dependencies
	deps := &models.HandlerDeps{
		Config: testConfig,
		Cache:  testCache,
		Logger: logging.NewLogger("INFO"),
	}

	// Create router
	router := NewRouter(deps)

	// Test that debug endpoints return 404 when disabled
	testCases := []struct {
		path       string
		statusCode int
	}{
		{"/debug/clear-cache", http.StatusNotFound},
		{"/debug/cache-dump", http.StatusNotFound},
		{"/debug/config", http.StatusNotFound},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest("GET", tc.path, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != tc.statusCode {
			t.Errorf("Expected status %d for %s, got %d", tc.statusCode, tc.path, rr.Code)
		}
	}
}
