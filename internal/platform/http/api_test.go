package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestAPIV1NotFoundUsesStandardErrorEnvelope(t *testing.T) {
	for _, path := range []string{APIV1Prefix, APIV1Prefix + "/missing"} {
		t.Run(path, func(t *testing.T) {
			response := serveAPIRequest(t, NewAPIV1Router(), http.MethodGet, path, "")

			assertAPIError(t, response, http.StatusNotFound, "route_not_found", "Route not found")
			if response.Header().Get(RequestIDHeader) == "" {
				t.Fatal("response request ID is empty")
			}
		})
	}
}

func TestAPIV1MethodNotAllowedUsesStandardErrorEnvelope(t *testing.T) {
	router := NewAPIV1Router()
	router.HandleFunc(http.MethodGet, "/probe", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	})

	response := serveAPIRequest(t, router, http.MethodPost, APIV1Prefix+"/probe", "")

	assertAPIError(t, response, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	if got := response.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("Allow = %q, want GET", got)
	}
}

func TestAPIV1PropagatesRequestIDToHandlerAndResponse(t *testing.T) {
	tests := []struct {
		name           string
		incoming       string
		want           string
		generatorCalls int
	}{
		{name: "accepted client ID", incoming: "desktop-request_123", want: "desktop-request_123"},
		{name: "missing ID", want: "generated-request-id", generatorCalls: 1},
		{name: "unsafe client ID", incoming: "bad/id", want: "generated-request-id", generatorCalls: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			generatorCalls := 0
			router := newAPIV1Router(func() (string, error) {
				generatorCalls++
				return "generated-request-id", nil
			})
			var contextRequestID string
			router.HandleFunc(http.MethodGet, "/probe", func(response http.ResponseWriter, request *http.Request) {
				contextRequestID = RequestIDFromContext(request.Context())
				response.WriteHeader(http.StatusNoContent)
			})

			response := serveAPIRequest(t, router, http.MethodGet, APIV1Prefix+"/probe", test.incoming)

			if response.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
			}
			if got := response.Header().Get(RequestIDHeader); got != test.want {
				t.Fatalf("response request ID = %q, want %q", got, test.want)
			}
			if contextRequestID != test.want {
				t.Fatalf("context request ID = %q, want %q", contextRequestID, test.want)
			}
			if generatorCalls != test.generatorCalls {
				t.Fatalf("generator calls = %d, want %d", generatorCalls, test.generatorCalls)
			}
		})
	}
}

func TestAPIV1HandlesRequestIDGenerationFailure(t *testing.T) {
	generationError := errors.New("random source unavailable")
	router := newAPIV1Router(func() (string, error) {
		return "", generationError
	})
	router.HandleFunc(http.MethodGet, "/probe", func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler called after request ID generation failure")
	})

	response := serveAPIRequest(t, router, http.MethodGet, APIV1Prefix+"/probe", "")

	assertAPIError(t, response, http.StatusInternalServerError, "internal_error", "Internal server error")
}

func TestGenerateRequestIDReturnsUUIDv4(t *testing.T) {
	requestID, err := generateRequestID()
	if err != nil {
		t.Fatalf("generateRequestID() error = %v", err)
	}
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !pattern.MatchString(requestID) {
		t.Fatalf("request ID = %q, want UUIDv4", requestID)
	}
}

func TestRequestIDFromContextWithoutMiddlewareIsEmpty(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Fatalf("RequestIDFromContext() = %q, want empty", got)
	}
}

func serveAPIRequest(
	t *testing.T,
	router *APIV1Router,
	method string,
	path string,
	requestID string,
) *httptest.ResponseRecorder {
	t.Helper()

	mux := http.NewServeMux()
	RegisterAPIV1(mux, router)
	request := httptest.NewRequest(method, path, nil)
	if requestID != "" {
		request.Header.Set(RequestIDHeader, requestID)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func assertAPIError(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantStatus int,
	wantCode string,
	wantMessage string,
) {
	t.Helper()

	if response.Code != wantStatus {
		t.Fatalf("status = %d, want %d", response.Code, wantStatus)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var envelope ErrorEnvelope
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if envelope.Error.Code != wantCode {
		t.Fatalf("error code = %q, want %q", envelope.Error.Code, wantCode)
	}
	if envelope.Error.Message != wantMessage {
		t.Fatalf("error message = %q, want %q", envelope.Error.Message, wantMessage)
	}
	if envelope.Error.Details == nil || len(envelope.Error.Details) != 0 {
		t.Fatalf("error details = %#v, want empty object", envelope.Error.Details)
	}
}
