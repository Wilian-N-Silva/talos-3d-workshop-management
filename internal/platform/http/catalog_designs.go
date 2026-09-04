package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	applicationcatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/catalog"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
	"github.com/go-chi/chi/v5"
)

const (
	CatalogPartsPath   = "/catalog/parts"
	DesignVersionsPath = "/catalog/design-versions"
	catalogDesignLimit = 64 * 1024
)

type CatalogDesignService interface {
	CreatePart(context.Context, string, domaincatalog.PartValues) (domaincatalog.Part, error)
	GetPart(context.Context, string) (domaincatalog.Part, error)
	ListParts(context.Context, string) ([]domaincatalog.Part, error)
	UpdatePart(context.Context, string, domaincatalog.PartValues) (domaincatalog.Part, error)
	DeletePart(context.Context, string) error
	CreateVersion(context.Context, string, string, domaincatalog.DesignVersionValues) (domaincatalog.DesignVersion, error)
	ListVersions(context.Context, string) ([]domaincatalog.DesignVersion, error)
	AttachFile(context.Context, string, string, string, domaincatalog.DesignFileRole) (domaincatalog.DesignFile, error)
}

type catalogPartRequest struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Notes    string `json:"notes"`
}

type catalogPartResponse struct {
	ID            string    `json:"id"`
	CatalogItemID string    `json:"catalog_item_id"`
	Name          string    `json:"name"`
	Quantity      int       `json:"quantity"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type designVersionRequest struct {
	Version              string                     `json:"version"`
	Notes                string                     `json:"notes"`
	Origin               domaincatalog.DesignOrigin `json:"origin"`
	SourceURL            *string                    `json:"source_url"`
	OriginalAuthor       string                     `json:"original_author"`
	LicenseName          string                     `json:"license_name"`
	CommercialUseAllowed *bool                      `json:"commercial_use_allowed"`
	AttributionRequired  bool                       `json:"attribution_required"`
	AttributionText      string                     `json:"attribution_text"`
}

type designFileRequest struct {
	FileID string                       `json:"file_id"`
	Role   domaincatalog.DesignFileRole `json:"role"`
}

type designFileResponse struct {
	FileID       string                       `json:"file_id"`
	Role         domaincatalog.DesignFileRole `json:"role"`
	OriginalName string                       `json:"original_name"`
	ContentType  string                       `json:"content_type"`
	SizeBytes    int64                        `json:"size_bytes"`
	SHA256       string                       `json:"sha256"`
	CreatedAt    time.Time                    `json:"created_at"`
}

type designVersionResponse struct {
	ID                   string                     `json:"id"`
	CatalogPartID        string                     `json:"catalog_part_id"`
	Version              string                     `json:"version"`
	Notes                string                     `json:"notes"`
	Origin               domaincatalog.DesignOrigin `json:"origin"`
	SourceURL            *string                    `json:"source_url"`
	OriginalAuthor       string                     `json:"original_author"`
	LicenseName          string                     `json:"license_name"`
	CommercialUseAllowed *bool                      `json:"commercial_use_allowed"`
	AttributionRequired  bool                       `json:"attribution_required"`
	AttributionText      string                     `json:"attribution_text"`
	CreatedBy            string                     `json:"created_by"`
	CreatedAt            time.Time                  `json:"created_at"`
	Files                []designFileResponse       `json:"files"`
}

func RegisterCatalogDesigns(router *APIV1Router, authentication BearerAuthenticationService, service CatalogDesignService) {
	itemPartsPath := CatalogItemsPath + "/{itemID}/parts"
	partPath := CatalogPartsPath + "/{partID}"
	partVersionsPath := partPath + "/design-versions"
	versionFilesPath := DesignVersionsPath + "/{versionID}/files"

	registerCatalogDesignRoute(router, authentication, domainauth.PermissionCatalogRead, http.MethodGet, itemPartsPath, func(response http.ResponseWriter, request *http.Request) {
		parts, err := service.ListParts(request.Context(), chi.URLParam(request, "itemID"))
		if err != nil {
			writeCatalogDesignError(response, err)
			return
		}
		result := make([]catalogPartResponse, 0, len(parts))
		for _, part := range parts {
			result = append(result, partResponse(part))
		}
		writeJSON(response, http.StatusOK, map[string]any{"parts": result})
	})
	registerCatalogDesignRoute(router, authentication, domainauth.PermissionCatalogWrite, http.MethodPost, itemPartsPath, func(response http.ResponseWriter, request *http.Request) {
		var body catalogPartRequest
		if !decodeCatalogDesignJSON(response, request, &body) {
			return
		}
		part, err := service.CreatePart(request.Context(), chi.URLParam(request, "itemID"), domaincatalog.PartValues{Name: body.Name, Quantity: body.Quantity, Notes: body.Notes})
		if err != nil {
			writeCatalogDesignError(response, err)
			return
		}
		writeJSON(response, http.StatusCreated, partResponse(part))
	})
	registerCatalogDesignRoute(router, authentication, domainauth.PermissionCatalogRead, http.MethodGet, partPath, func(response http.ResponseWriter, request *http.Request) {
		part, err := service.GetPart(request.Context(), chi.URLParam(request, "partID"))
		if err != nil {
			writeCatalogDesignError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, partResponse(part))
	})
	registerCatalogDesignRoute(router, authentication, domainauth.PermissionCatalogWrite, http.MethodPut, partPath, func(response http.ResponseWriter, request *http.Request) {
		var body catalogPartRequest
		if !decodeCatalogDesignJSON(response, request, &body) {
			return
		}
		part, err := service.UpdatePart(request.Context(), chi.URLParam(request, "partID"), domaincatalog.PartValues{Name: body.Name, Quantity: body.Quantity, Notes: body.Notes})
		if err != nil {
			writeCatalogDesignError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, partResponse(part))
	})
	registerCatalogDesignRoute(router, authentication, domainauth.PermissionCatalogWrite, http.MethodDelete, partPath, func(response http.ResponseWriter, request *http.Request) {
		if err := service.DeletePart(request.Context(), chi.URLParam(request, "partID")); err != nil {
			writeCatalogDesignError(response, err)
			return
		}
		response.WriteHeader(http.StatusNoContent)
	})
	registerCatalogDesignRoute(router, authentication, domainauth.PermissionCatalogRead, http.MethodGet, partVersionsPath, func(response http.ResponseWriter, request *http.Request) {
		versions, err := service.ListVersions(request.Context(), chi.URLParam(request, "partID"))
		if err != nil {
			writeCatalogDesignError(response, err)
			return
		}
		result := make([]designVersionResponse, 0, len(versions))
		for _, version := range versions {
			result = append(result, versionResponse(version))
		}
		writeJSON(response, http.StatusOK, map[string]any{"versions": result})
	})
	registerCatalogDesignRoute(router, authentication, domainauth.PermissionCatalogWrite, http.MethodPost, partVersionsPath, func(response http.ResponseWriter, request *http.Request) {
		var body designVersionRequest
		if !decodeCatalogDesignJSON(response, request, &body) {
			return
		}
		user, ok := CurrentUserFromContext(request.Context())
		if !ok {
			writeUnauthenticated(response)
			return
		}
		version, err := service.CreateVersion(request.Context(), chi.URLParam(request, "partID"), user.ID, domaincatalog.DesignVersionValues{
			Version: body.Version, Notes: body.Notes, Origin: body.Origin, SourceURL: body.SourceURL,
			OriginalAuthor: body.OriginalAuthor, LicenseName: body.LicenseName, CommercialUseAllowed: body.CommercialUseAllowed,
			AttributionRequired: body.AttributionRequired, AttributionText: body.AttributionText,
		})
		if err != nil {
			writeCatalogDesignError(response, err)
			return
		}
		writeJSON(response, http.StatusCreated, versionResponse(version))
	})
	registerCatalogDesignRoute(router, authentication, domainauth.PermissionCatalogWrite, http.MethodPost, versionFilesPath, func(response http.ResponseWriter, request *http.Request) {
		var body designFileRequest
		if !decodeCatalogDesignJSON(response, request, &body) {
			return
		}
		user, ok := CurrentUserFromContext(request.Context())
		if !ok {
			writeUnauthenticated(response)
			return
		}
		file, err := service.AttachFile(request.Context(), chi.URLParam(request, "versionID"), body.FileID, user.ID, body.Role)
		if err != nil {
			writeCatalogDesignError(response, err)
			return
		}
		writeJSON(response, http.StatusCreated, designFileResponseFrom(file))
	})
}

func registerCatalogDesignRoute(router *APIV1Router, authentication BearerAuthenticationService, permission domainauth.Permission, method, path string, handler http.HandlerFunc) {
	router.Handle(method, path, RequirePermission(authentication, permission)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		handler(response, request)
	})))
}

func decodeCatalogDesignJSON(response http.ResponseWriter, request *http.Request, target any) bool {
	if request.Header.Get("Content-Type") != "application/json" {
		WriteError(response, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json", nil)
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, catalogDesignLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		WriteError(response, http.StatusBadRequest, "invalid_catalog_design", "Invalid catalog design data", nil)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		WriteError(response, http.StatusBadRequest, "invalid_catalog_design", "Invalid catalog design data", nil)
		return false
	}
	return true
}

func writeCatalogDesignError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, applicationcatalog.ErrInvalidPart), errors.Is(err, applicationcatalog.ErrInvalidDesignVersion), errors.Is(err, applicationcatalog.ErrInvalidDesignFile):
		WriteError(response, http.StatusBadRequest, "invalid_catalog_design", "Invalid catalog design data", nil)
	case errors.Is(err, domaincatalog.ErrItemNotFound):
		WriteError(response, http.StatusNotFound, "catalog_item_not_found", "Catalog item not found", nil)
	case errors.Is(err, domaincatalog.ErrPartNotFound):
		WriteError(response, http.StatusNotFound, "catalog_part_not_found", "Catalog part not found", nil)
	case errors.Is(err, domaincatalog.ErrDesignVersionNotFound), errors.Is(err, domaincatalog.ErrDesignFileNotFound):
		WriteError(response, http.StatusNotFound, "design_reference_not_found", "Design version or file not found", nil)
	case errors.Is(err, domaincatalog.ErrDesignVersionConflict):
		WriteError(response, http.StatusConflict, "design_version_exists", "Design version already exists", nil)
	case errors.Is(err, domaincatalog.ErrDesignFileConflict):
		WriteError(response, http.StatusConflict, "design_file_exists", "Design file link already exists", nil)
	case errors.Is(err, domaincatalog.ErrDesignHistoryExists):
		WriteError(response, http.StatusConflict, "design_history_exists", "Immutable design history prevents deletion", nil)
	default:
		WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
	}
}

func partResponse(part domaincatalog.Part) catalogPartResponse {
	return catalogPartResponse{ID: part.ID, CatalogItemID: part.CatalogItemID, Name: part.Name, Quantity: part.Quantity, Notes: part.Notes, CreatedAt: part.CreatedAt, UpdatedAt: part.UpdatedAt}
}

func versionResponse(version domaincatalog.DesignVersion) designVersionResponse {
	files := make([]designFileResponse, 0, len(version.Files))
	for _, file := range version.Files {
		files = append(files, designFileResponseFrom(file))
	}
	return designVersionResponse{ID: version.ID, CatalogPartID: version.CatalogPartID, Version: version.Version, Notes: version.Notes,
		Origin: version.Origin, SourceURL: version.SourceURL, OriginalAuthor: version.OriginalAuthor, LicenseName: version.LicenseName,
		CommercialUseAllowed: version.CommercialUseAllowed, AttributionRequired: version.AttributionRequired, AttributionText: version.AttributionText,
		CreatedBy: version.CreatedBy, CreatedAt: version.CreatedAt, Files: files}
}

func designFileResponseFrom(file domaincatalog.DesignFile) designFileResponse {
	return designFileResponse{FileID: file.FileID, Role: file.Role, OriginalName: file.OriginalName, ContentType: file.ContentType, SizeBytes: file.SizeBytes, SHA256: file.SHA256Hex, CreatedAt: file.CreatedAt}
}
