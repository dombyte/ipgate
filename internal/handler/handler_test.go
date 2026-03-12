package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dombyte/ipgate/internal/cache"
	"github.com/dombyte/ipgate/internal/config"
	"github.com/dombyte/ipgate/internal/ipmatcher"
	"github.com/dombyte/ipgate/internal/logging"
	"github.com/dombyte/ipgate/internal/models"
)

func createTestDeps() *models.HandlerDeps {
	cfg := &config.Config{}
	logger := logging.NewLogger("INFO")
	return &models.HandlerDeps{
		Config: cfg,
		Cache:  cache.NewCache(300, 100000, 60, 64, false, 0),
		Logger: logger,
	}
}

func TestBouncerHandler(t *testing.T) {
	deps := createTestDeps()
	
	// Create a test IPMatcher
	matcher := ipmatcher.NewIPMatcher()
	deps.Config.IPMatcher = matcher
	
	// Load blocklist
	if err := matcher.LoadBlocklist([]string{"198.51.100.1"}); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}
	
	// Set up headers
	deps.Config.Headers = map[string]string{
		"client_ip_header": "X-Forwarded-For",
		"host_header":     "Host",
		"uri_header":      "Request-URI",
		"method_header":   "Method",
		"proto_header":    "Proto",
	}
	deps.Config.ErrorFormat = "json"
	deps.Config.StatusDenied = http.StatusForbidden
	deps.Config.StatusAllowed = http.StatusOK
	
	handler := BouncerHandler(deps)
	
	// Test blocked IP
	req := httptest.NewRequest("GET", "/bouncer/test", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for blocked IP, got %d", w.Code)
	}
	
	// Test allowed IP
	req2 := httptest.NewRequest("GET", "/bouncer/test", nil)
	req2.Header.Set("X-Forwarded-For", "192.168.1.1")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	
	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for allowed IP, got %d", w2.Code)
	}
}

func TestBouncerHandler_NoClientIP(t *testing.T) {
	deps := createTestDeps()
	deps.Config.Headers = map[string]string{
		"client_ip_header": "X-Forwarded-For",
	}
	
	handler := BouncerHandler(deps)
	
	req := httptest.NewRequest("GET", "/bouncer/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing client IP, got %d", w.Code)
	}
}

func TestBouncerHandler_InvalidIP(t *testing.T) {
	deps := createTestDeps()
	deps.Config.Headers = map[string]string{
		"client_ip_header": "X-Forwarded-For",
	}
	matcher := ipmatcher.NewIPMatcher()
	deps.Config.IPMatcher = matcher
	
	handler := BouncerHandler(deps)
	
	req := httptest.NewRequest("GET", "/bouncer/test", nil)
	req.Header.Set("X-Forwarded-For", "invalid-ip")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid IP, got %d", w.Code)
	}
}

func TestBouncerHandler_WithCache(t *testing.T) {
	deps := createTestDeps()
	deps.Config.Headers = map[string]string{
		"client_ip_header": "X-Forwarded-For",
	}
	matcher := ipmatcher.NewIPMatcher()
	deps.Config.IPMatcher = matcher
	if err := matcher.LoadBlocklist([]string{"198.51.100.1"}); err != nil {
		t.Fatalf("Failed to load blocklist: %v", err)
	}
	deps.Config.ErrorFormat = "json"
	deps.Config.StatusDenied = http.StatusForbidden
	deps.Config.StatusAllowed = http.StatusOK
	
	// Pre-populate cache with DENY
	deps.Cache.Add("198.51.100.1", "DENY")
	
	handler := BouncerHandler(deps)
	
	req := httptest.NewRequest("GET", "/bouncer/test", nil)
	req.Header.Set("X-Forwarded-For", "198.51.100.1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for cached DENY, got %d", w.Code)
	}
	
	// Test cached ALLOW
	deps.Cache.Add("192.168.1.1", "ALLOW")
	
	req2 := httptest.NewRequest("GET", "/bouncer/test", nil)
	req2.Header.Set("X-Forwarded-For", "192.168.1.1")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	
	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for cached ALLOW, got %d", w2.Code)
	}
}

func TestHealthHandler(t *testing.T) {
	handler := HealthHandler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that the response contains the expected JSON
	body := w.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("Expected response to contain status:ok, got: %s", body)
	}
}

func TestClearCacheHandler(t *testing.T) {
	deps := createTestDeps()
	deps.Cache.Add("198.51.100.1", cache.CacheDeny)
	handler := ClearCacheHandler(deps)

	// Test clearing cache
	req := httptest.NewRequest("POST", "/debug/clear-cache", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify cache is cleared
	status := deps.Cache.GetStatus("198.51.100.1")
	if status != cache.CacheMiss {
		t.Errorf("Expected CacheMiss after clear, got %s", status)
	}
}

func TestCacheDumpHandler(t *testing.T) {
	deps := createTestDeps()
	deps.Cache.Add("198.51.100.1", cache.CacheDeny)
	handler := CacheDumpHandler(deps)

	req := httptest.NewRequest("GET", "/debug/cache-dump", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestConfigHandler(t *testing.T) {
	deps := createTestDeps()
	handler := ConfigHandler(deps)

	req := httptest.NewRequest("GET", "/debug/config", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
