package httpplatform

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

const APIV1Prefix = "/api/v1"

// APIV1Router owns versioned API routes and their shared HTTP contracts.
type APIV1Router struct {
	router  chi.Router
	handler http.Handler
}

// NewAPIV1Router creates an empty v1 router with standardized errors and
// request-ID propagation.
func NewAPIV1Router() *APIV1Router {
	return newAPIV1Router(generateRequestID)
}

func newAPIV1Router(generator requestIDGenerator) *APIV1Router {
	router := newJSONRouter()

	return &APIV1Router{
		router:  router,
		handler: requestIDMiddleware(generator)(router),
	}
}

func newJSONRouter() chi.Router {
	router := chi.NewRouter()
	router.NotFound(func(response http.ResponseWriter, _ *http.Request) {
		WriteError(response, http.StatusNotFound, "route_not_found", "Route not found", nil)
	})
	router.MethodNotAllowed(func(response http.ResponseWriter, request *http.Request) {
		for _, method := range allowedMethods(router, request.URL.Path) {
			response.Header().Add("Allow", method)
		}
		WriteError(response, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", nil)
	})

	return router
}

// Handle registers a method-specific route relative to /api/v1.
func (router *APIV1Router) Handle(method, pattern string, handler http.Handler) {
	router.router.Method(method, pattern, handler)
}

// HandleFunc registers a method-specific handler function relative to /api/v1.
func (router *APIV1Router) HandleFunc(method, pattern string, handler http.HandlerFunc) {
	router.Handle(method, pattern, handler)
}

func (router *APIV1Router) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	router.handler.ServeHTTP(response, request)
}

func allowedMethods(router chi.Routes, path string) []string {
	methods := []string{
		http.MethodConnect,
		http.MethodDelete,
		http.MethodGet,
		http.MethodHead,
		http.MethodOptions,
		http.MethodPatch,
		http.MethodPost,
		http.MethodPut,
		http.MethodTrace,
	}
	allowed := make([]string, 0, len(methods))
	for _, method := range methods {
		if router.Match(chi.NewRouteContext(), method, path) {
			allowed = append(allowed, method)
		}
	}
	return allowed
}

// RegisterAPIV1 mounts a v1 router without redirecting the exact prefix.
func RegisterAPIV1(mux *http.ServeMux, router *APIV1Router) {
	handler := http.StripPrefix(APIV1Prefix, router)
	mux.Handle(APIV1Prefix, handler)
	mux.Handle(APIV1Prefix+"/", handler)
}
