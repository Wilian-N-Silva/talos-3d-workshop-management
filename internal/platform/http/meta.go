package httpplatform

import (
	"encoding/json"
	"net/http"
)

const (
	// APIVersion identifies the versioned HTTP contract exposed by this router.
	APIVersion = "v1"
	// MetaPath is relative to APIV1Prefix.
	MetaPath = "/meta"
)

// MetaResponse describes server and desktop compatibility metadata.
type MetaResponse struct {
	APIVersion            string `json:"api_version"`
	ServerVersion         string `json:"server_version"`
	WorkshopName          string `json:"workshop_name"`
	MinimumDesktopVersion string `json:"minimum_desktop_version"`
}

// RegisterMeta registers the unauthenticated server metadata endpoint.
func RegisterMeta(router *APIV1Router, metadata MetaResponse) {
	router.HandleFunc(http.MethodGet, MetaPath, func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(metadata)
	})
}
