package httpplatform

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"

	applicationsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/settings"
	applicationstorage "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/storage"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
)

const (
	WorkshopLogoUploadPath   = "/settings/logo"
	WorkshopLogoDownloadPath = "/meta/logo"
	multipartOverheadBytes   = 1024 * 1024
)

// WorkshopLogoService stores and opens the currently associated logo.
type WorkshopLogoService interface {
	Upload(context.Context, applicationsettings.LogoUpload) (applicationsettings.LogoUploadResult, error)
	OpenCurrent(context.Context) (applicationsettings.LogoDownload, error)
}

type workshopLogoResponse struct {
	FileID  string `json:"file_id"`
	LogoURL string `json:"logo_url"`
}

// RegisterWorkshopLogo registers privileged upload and association-authorized
// public download for login branding.
func RegisterWorkshopLogo(
	router *APIV1Router,
	authentication BearerAuthenticationService,
	service WorkshopLogoService,
	maximumBytes int64,
) {
	router.Handle(http.MethodPost, WorkshopLogoUploadPath, RequirePermission(
		authentication,
		domainauth.PermissionSettingsManage,
	)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		user, ok := CurrentUserFromContext(request.Context())
		if !ok {
			writeUnauthenticated(response)
			return
		}
		request.Body = http.MaxBytesReader(response, request.Body, maximumBytes+multipartOverheadBytes)
		if err := request.ParseMultipartForm(maximumBytes); err != nil {
			WriteError(response, http.StatusBadRequest, "invalid_logo", "Invalid workshop logo", nil)
			return
		}
		if request.MultipartForm != nil {
			defer request.MultipartForm.RemoveAll()
		}
		file, header, err := request.FormFile("file")
		if err != nil {
			WriteError(response, http.StatusBadRequest, "invalid_logo", "Invalid workshop logo", nil)
			return
		}
		defer file.Close()

		result, err := service.Upload(request.Context(), applicationsettings.LogoUpload{
			UploadedBy:   user.ID,
			OriginalName: header.Filename,
			Content:      file,
		})
		switch {
		case errors.Is(err, applicationsettings.ErrInvalidWorkshopLogo):
			WriteError(response, http.StatusBadRequest, "invalid_logo", "Invalid workshop logo", nil)
			return
		case err != nil:
			WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			return
		}
		writeJSON(response, http.StatusCreated, workshopLogoResponse{
			FileID:  result.File.ID,
			LogoURL: APIV1Prefix + WorkshopLogoDownloadPath,
		})
	})))

	router.HandleFunc(http.MethodGet, WorkshopLogoDownloadPath, func(response http.ResponseWriter, request *http.Request) {
		download, err := service.OpenCurrent(request.Context())
		switch {
		case errors.Is(err, domainfiles.ErrFileNotFound), errors.Is(err, applicationstorage.ErrObjectNotFound):
			WriteError(response, http.StatusNotFound, "logo_not_found", "Workshop logo not found", nil)
			return
		case err != nil:
			WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			return
		}
		defer download.Content.Close()

		response.Header().Set("Cache-Control", "no-cache")
		response.Header().Set("Content-Type", download.File.ContentType)
		response.Header().Set("Content-Length", strconv.FormatInt(download.File.SizeBytes, 10))
		response.Header().Set("ETag", `"sha256-`+hex.EncodeToString(download.File.SHA256)+`"`)
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.WriteHeader(http.StatusOK)
		_, _ = io.Copy(response, download.Content)
	})
}
