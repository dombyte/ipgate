package models

import (
	"net/http/httptest"
	"testing"
)

func TestResponseWriter(t *testing.T) {
	t.Run("WriteHeader", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		writer := &ResponseWriter{ResponseWriter: recorder}

		writer.WriteHeader(403)

		if writer.StatusCode != 403 {
			t.Errorf("Expected status code 403, got %d", writer.StatusCode)
		}

		if recorder.Code != 403 {
			t.Errorf("Expected recorder code 403, got %d", recorder.Code)
		}
	})

	t.Run("Write", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		writer := &ResponseWriter{ResponseWriter: recorder}

		data := []byte("test data")
		n, err := writer.Write(data)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if n != len(data) {
			t.Errorf("Expected to write %d bytes, got %d", len(data), n)
		}

		if writer.BytesWritten != len(data) {
			t.Errorf("Expected BytesWritten to be %d, got %d", len(data), writer.BytesWritten)
		}

		if recorder.Body.String() != "test data" {
			t.Errorf("Expected body 'test data', got '%s'", recorder.Body.String())
		}
	})

	t.Run("WriteHeaderAndWrite", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		writer := &ResponseWriter{ResponseWriter: recorder}

		writer.WriteHeader(200)
		writer.Write([]byte("response"))

		if writer.StatusCode != 200 {
			t.Errorf("Expected status code 200, got %d", writer.StatusCode)
		}

		if writer.BytesWritten != 8 {
			t.Errorf("Expected BytesWritten to be 8, got %d", writer.BytesWritten)
		}
	})
}

func TestErrorPageData(t *testing.T) {
	data := ErrorPageData{
		RequestID:   "req-123",
		ClientIP:    "198.51.100.1",
		Host:        "example.com",
		URI:         "/test",
		Method:      "GET",
		Proto:       "HTTP/1.1",
		Headers:     map[string]string{"User-Agent": "test"},
		StatusCode:  403,
		BlockReason: "blocked",
	}

	if data.RequestID != "req-123" {
		t.Error("RequestID not set correctly")
	}
	if data.ClientIP != "198.51.100.1" {
		t.Error("ClientIP not set correctly")
	}
}

func TestHealthResponse(t *testing.T) {
	resp := HealthResponse{Status: "ok"}

	if resp.Status != "ok" {
		t.Error("Status not set correctly")
	}
}

func TestCacheEntry(t *testing.T) {
	entry := CacheEntry{
		Status:    "DENY",
		Timestamp: 1234567890,
	}

	if entry.Status != "DENY" {
		t.Error("Status not set correctly")
	}
	if entry.Timestamp != 1234567890 {
		t.Error("Timestamp not set correctly")
	}
}

func TestCacheDumpResponse(t *testing.T) {
	resp := CacheDumpResponse{
		Entries: map[string]CacheEntry{
			"198.51.100.1": {Status: "DENY", Timestamp: 1234567890},
			"203.0.113.1":  {Status: "ALLOW", Timestamp: 1234567891},
		},
	}

	if len(resp.Entries) != 2 {
		t.Error("Entries not set correctly")
	}

	entry, exists := resp.Entries["198.51.100.1"]
	if !exists {
		t.Error("Entry for 198.51.100.1 not found")
	}
	if entry.Status != "DENY" {
		t.Error("Entry status not set correctly")
	}
}
