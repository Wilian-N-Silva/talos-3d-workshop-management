// Package httpplatform contains server HTTP transport concerns.
package httpplatform

import (
	"context"
	"io"
	"net/http"
)

const (
	// LivenessPath is the dependency-free process liveness endpoint.
	LivenessPath = "/health/live"
	// ReadinessPath reports whether required server dependencies are available.
	ReadinessPath       = "/health/ready"
	okResponse          = "{\"status\":\"ok\"}\n"
	unavailableResponse = "{\"status\":\"unavailable\"}\n"
)

// ReadinessChecker checks the dependencies required to serve requests.
type ReadinessChecker interface {
	Check(context.Context) error
}

// RegisterLiveness registers the process-only liveness endpoint. It accepts no
// infrastructure dependencies so database, storage, and printer state cannot
// affect the response.
func RegisterLiveness(mux *http.ServeMux) {
	mux.HandleFunc("GET "+LivenessPath, liveness)
}

// RegisterReadiness registers the dependency-aware readiness endpoint.
func RegisterReadiness(mux *http.ServeMux, checker ReadinessChecker) {
	mux.HandleFunc("GET "+ReadinessPath, func(response http.ResponseWriter, request *http.Request) {
		readiness(response, request, checker)
	})
}

func liveness(response http.ResponseWriter, _ *http.Request) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(response, okResponse)
}

func readiness(response http.ResponseWriter, request *http.Request, checker ReadinessChecker) {
	status := http.StatusOK
	body := okResponse
	if err := checker.Check(request.Context()); err != nil {
		status = http.StatusServiceUnavailable
		body = unavailableResponse
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_, _ = io.WriteString(response, body)
}
