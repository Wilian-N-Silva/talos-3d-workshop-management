package httpplatform

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestMetaReturnsConfiguredMetadata(t *testing.T) {
	want := MetaResponse{
		APIVersion:            APIVersion,
		ServerVersion:         "1.2.3",
		WorkshopName:          "Prototype Lab",
		MinimumDesktopVersion: "1.1.0",
	}
	router := NewAPIV1Router()
	RegisterMeta(router, want)

	response := serveAPIRequest(t, router, http.MethodGet, APIV1Prefix+MetaPath, "meta-request")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := response.Header().Get(RequestIDHeader); got != "meta-request" {
		t.Fatalf("X-Request-ID = %q, want meta-request", got)
	}

	var got MetaResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if got != want {
		t.Fatalf("metadata = %#v, want %#v", got, want)
	}
}

func TestMetaRejectsUnsupportedMethods(t *testing.T) {
	router := NewAPIV1Router()
	RegisterMeta(router, MetaResponse{})

	response := serveAPIRequest(t, router, http.MethodPost, APIV1Prefix+MetaPath, "")

	assertAPIError(t, response, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
}
