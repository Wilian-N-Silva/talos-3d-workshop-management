package httpplatform

import "net/http"

const (
	// APIVersion identifies the versioned HTTP contract exposed by this router.
	APIVersion = "v1"
	// MetaPath is relative to APIV1Prefix.
	MetaPath = "/meta"
)

// MetaResponse describes server and desktop compatibility metadata.
type MetaResponse struct {
	APIVersion            string  `json:"api_version"`
	ServerVersion         string  `json:"server_version"`
	WorkshopName          string  `json:"workshop_name"`
	LogoURL               *string `json:"logo_url"`
	MinimumDesktopVersion string  `json:"minimum_desktop_version"`
}

// RegisterMeta registers the unauthenticated server metadata endpoint with the
// current persisted workshop name.
func RegisterMeta(router *APIV1Router, metadata MetaResponse, settings WorkshopSettingsReader) {
	router.HandleFunc(http.MethodGet, MetaPath, func(response http.ResponseWriter, request *http.Request) {
		current, err := settings.Get(request.Context())
		if err != nil {
			WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			return
		}
		responseMetadata := metadata
		responseMetadata.WorkshopName = current.WorkshopName
		responseMetadata.LogoURL = workshopLogoURL(current.LogoFileID)
		writeJSON(response, http.StatusOK, responseMetadata)
	})
}
