package httpplatform

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type readinessStub struct {
	err   error
	calls int
}

func (stub *readinessStub) Check(context.Context) error {
	stub.calls++
	return stub.err
}

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

func TestReadinessReturnsStableJSON(t *testing.T) {
	tests := []struct {
		name       string
		checkError error
		wantStatus int
		wantBody   string
	}{
		{name: "ready", wantStatus: http.StatusOK, wantBody: "{\"status\":\"ok\"}\n"},
		{
			name:       "unavailable",
			checkError: errors.New("dependency unavailable"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "{\"status\":\"unavailable\"}\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			checker := &readinessStub{err: test.checkError}
			mux := http.NewServeMux()
			RegisterReadiness(mux, checker)

			request := httptest.NewRequest(http.MethodGet, ReadinessPath, nil)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if got := response.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", got)
			}
			if got := response.Body.String(); got != test.wantBody {
				t.Fatalf("body = %q, want %q", got, test.wantBody)
			}
			if checker.calls != 1 {
				t.Fatalf("readiness checker calls = %d, want 1", checker.calls)
			}
		})
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
