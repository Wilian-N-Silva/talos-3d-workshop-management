// Package httpplatform contains server HTTP transport concerns.
package httpplatform

import (
	"io"
	"net/http"
)

const (
	// LivenessPath is the dependency-free process liveness endpoint.
	LivenessPath = "/health/live"
	liveResponse = "{\"status\":\"ok\"}\n"
)

// RegisterLiveness registers the process-only liveness endpoint. It accepts no
// infrastructure dependencies so database, storage, and printer state cannot
// affect the response.
func RegisterLiveness(mux *http.ServeMux) {
	mux.HandleFunc("GET "+LivenessPath, liveness)
}

func liveness(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(response, liveResponse)
}
