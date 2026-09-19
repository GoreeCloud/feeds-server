package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCapabilities(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/capabilities", nil)
	res := httptest.NewRecorder()

	NewHandler().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
	}
	if got := res.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("unexpected content type %q", got)
	}
	if got := res.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("unexpected cache policy %q", got)
	}

	var body capabilityResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Product != ProductName {
		t.Fatalf("unexpected product %q", body.Product)
	}
	if body.APIVersion != APIVersion {
		t.Fatalf("unexpected API version %q", body.APIVersion)
	}
	if body.ProtocolVersion != ProtocolVersion {
		t.Fatalf("unexpected protocol version %q", body.ProtocolVersion)
	}
	if body.Lifecycle != Lifecycle {
		t.Fatalf("unexpected lifecycle %q", body.Lifecycle)
	}
	if body.Capabilities == nil {
		t.Fatal("capabilities must be an empty array, not null")
	}
	if len(body.Capabilities) != 0 {
		t.Fatalf("expected no optional capabilities, got %v", body.Capabilities)
	}
}

func TestHealthEndpoints(t *testing.T) {
	for _, path := range []string{"/health/live", "/health/ready"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			res := httptest.NewRecorder()

			NewHandler().ServeHTTP(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("expected %d, got %d", http.StatusOK, res.Code)
			}

			var body healthResponse
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Status != "ok" {
				t.Fatalf("unexpected health status %q", body.Status)
			}
		})
	}
}

func TestUnsupportedMethodIsRejected(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/capabilities", nil)
	res := httptest.NewRecorder()

	NewHandler().ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, res.Code)
	}
}

func TestUnknownPathIsNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	res := httptest.NewRecorder()

	NewHandler().ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected %d, got %d", http.StatusNotFound, res.Code)
	}
}
