package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dombyte/ipgate/internal/cache"
	"github.com/dombyte/ipgate/internal/config"
	"github.com/dombyte/ipgate/internal/ipmatcher"
	"github.com/dombyte/ipgate/internal/logging"
	"github.com/dombyte/ipgate/internal/models"
)

func TestNewRouter(t *testing.T) {
	// Create IPMatcher and load data
	matcher := ipmatcher.NewIPMatcher()
	if err := matcher.LoadBlocklist([]string{"198.51.100.1", "10.0.0.0/24"}); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}
	if err := matcher.LoadWhitelist([]string{"198.51.100.1", "203.0.113.0/24"}); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	// Create test config
	testConfig := &config.Config{
		Port:             "8080",
		StatusAllowed:    200,
		StatusDenied:     403,
		ErrorFormat:      "json",
		ErrorPage:        "templates/error.html",
		BlocklistMaxSize: 1024 * 1024,
		Headers: map[string]string{
			"client_ip_header": "X-Forwarded-For",
		},
		IPMatcher: matcher,
	}

	// Create test cache
	testCache := cache.NewCache(300, 10000, 60, 64, false, 0)

	// Create handler dependencies
	handlerDeps := &models.HandlerDeps{
		Config: testConfig,
		Cache:  testCache,
		Logger: logging.NewLogger("INFO"),
	}

	// Create router
	router := NewRouter(handlerDeps)

	// Test health endpoint
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for /health, got %d", rr.Code)
	}

	// Test bouncer endpoint with blocked IP
	req, _ = http.NewRequest("GET", "/bouncer", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1") // This IP is in blocklist but not whitelist
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 403 {
		t.Errorf("Expected status 403 for blocked IP, got %d", rr.Code)
	}

	// Test bouncer endpoint with whitelisted IP
	req, _ = http.NewRequest("GET", "/bouncer", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.1") // This IP is whitelisted
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Errorf("Expected status 200 for whitelisted IP, got %d", rr.Code)
	}
}

func TestNewRouterWithRateLimiting(t *testing.T) {
	// Create IPMatcher and load data
	matcher := ipmatcher.NewIPMatcher()
	if err := matcher.LoadBlocklist([]string{"198.51.100.1", "10.0.0.0/24"}); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}
	if err := matcher.LoadWhitelist([]string{"198.51.100.1", "203.0.113.0/24"}); err != nil {
		t.Fatalf("Failed to load whitelist: %v", err)
	}

	// Create test config
	testConfig := &config.Config{
		Port:             "8080",
		StatusAllowed:    200,
		StatusDenied:     403,
		ErrorFormat:      "json",
		ErrorPage:        "templates/error.html",
		BlocklistMaxSize: 1024 * 1024,
		Headers: map[string]string{
			"client_ip_header": "X-Forwarded-For",
		},
		IPMatcher: matcher,
	}

	// Create test cache
	testCache := cache.NewCache(300, 10000, 60, 64, false, 0)

	// Create handler dependencies
	handlerDeps := &models.HandlerDeps{
		Config: testConfig,
		Cache:  testCache,
	}

	// Create router
	router := NewRouter(handlerDeps)

	// Test that rate limiting middleware is applied
	req, _ := http.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for /health, got %d", rr.Code)
	}
}
