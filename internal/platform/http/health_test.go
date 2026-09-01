package httpplatform

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLivenessReturnsStableJSONWithoutDependencies(t *testing.T) {
	mux := http.NewServeMux()
	RegisterLiveness(mux)

	request := httptest.NewRequest(http.MethodGet, LivenessPath, nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := response.Body.String(); got != "{\"status\":\"ok\"}\n" {
		t.Fatalf("body = %q, want stable liveness JSON", got)
	}
}

func TestLivenessRejectsUnsupportedMethods(t *testing.T) {
	mux := http.NewServeMux()
	RegisterLiveness(mux)

	request := httptest.NewRequest(http.MethodPost, LivenessPath, nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
