package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dombyte/ipgate/internal/cache"
	"github.com/dombyte/ipgate/internal/config"
	"github.com/dombyte/ipgate/internal/models"
)

func createTestDeps() *models.HandlerDeps {
	cfg := &config.Config{}
	return &models.HandlerDeps{
		Config: cfg,
		Cache:  cache.NewCache(300, 100000, 60, 64, false, 0),
	}
}

func TestRequestLoggingMiddleware(t *testing.T) {
	deps := createTestDeps()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with middleware
	middleware := RequestLoggingMiddleware(deps)
	wrappedHandler := middleware(testHandler)

	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "198.51.100.1:12345"
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRequestLoggingMiddleware_Concurrent(t *testing.T) {
	deps := createTestDeps()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestLoggingMiddleware(deps)
	wrappedHandler := middleware(testHandler)

	// Test concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "198.51.100.1:12345"
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)
			done <- true
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRequestLoggingMiddleware_DifferentMethods(t *testing.T) {
	deps := createTestDeps()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestLoggingMiddleware(deps)
	wrappedHandler := middleware(testHandler)

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		req := httptest.NewRequest(method, "/test", nil)
		req.RemoteAddr = "198.51.100.1:12345"
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for %s, got %d", method, w.Code)
		}
	}
}

func TestRequestLoggingMiddleware_WithQueryParams(t *testing.T) {
	deps := createTestDeps()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestLoggingMiddleware(deps)
	wrappedHandler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test?param1=value1&param2=value2", nil)
	req.RemoteAddr = "198.51.100.1:12345"
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRequestLoggingMiddleware_WithHeaders(t *testing.T) {
	deps := createTestDeps()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestLoggingMiddleware(deps)
	wrappedHandler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "198.51.100.1:12345"
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRequestLoggingMiddleware_WithBody(t *testing.T) {
	deps := createTestDeps()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestLoggingMiddleware(deps)
	wrappedHandler := middleware(testHandler)

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "198.51.100.1:12345"
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRequestLoggingMiddleware_IPv6(t *testing.T) {
	deps := createTestDeps()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestLoggingMiddleware(deps)
	wrappedHandler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "[2001:db8::1]:12345"
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRequestLoggingMiddleware_NoLogger(t *testing.T) {
	deps := createTestDeps()
	// Don't set logger

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestLoggingMiddleware(deps)
	wrappedHandler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "198.51.100.1:12345"
	w := httptest.NewRecorder()

	// Should not panic even without logger
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRequestLoggingMiddleware_EmptyRemoteAddr(t *testing.T) {
	deps := createTestDeps()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestLoggingMiddleware(deps)
	wrappedHandler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	// Empty RemoteAddr
	w := httptest.NewRecorder()

	// Should not panic with empty RemoteAddr
	wrappedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
