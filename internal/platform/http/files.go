package httpplatform

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	applicationfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/files"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"

	"github.com/go-chi/chi/v5"
)

const (
	FilesPath             = "/files"
	fileDownloadPath      = FilesPath + "/{fileID}"
	fileMultipartOverhead = 1024 * 1024
	fileMultipartMemory   = 1024 * 1024
	maximumInt64          = 1<<63 - 1
)

// FileTransferService stores uploads and opens authorized downloads.
type FileTransferService interface {
	UploadFile(context.Context, applicationfiles.Upload) (applicationfiles.UploadResult, error)
	OpenFile(context.Context, string) (applicationfiles.Download, error)
}

type fileResponse struct {
	ID           string    `json:"id"`
	SHA256       string    `json:"sha256"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	UploadedBy   string    `json:"uploaded_by"`
	CreatedAt    time.Time `json:"created_at"`
	Deduplicated bool      `json:"deduplicated"`
}

// RegisterFiles registers permission-protected immutable upload and download.
func RegisterFiles(
	router *APIV1Router,
	authentication BearerAuthenticationService,
	service FileTransferService,
	maximumBytes int64,
) {
	router.Handle(http.MethodPost, FilesPath, RequirePermission(
		authentication,
		domainauth.PermissionFilesUpload,
	)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		user, ok := CurrentUserFromContext(request.Context())
		if !ok {
			writeUnauthenticated(response)
			return
		}
		bodyLimit := maximumBytes
		if maximumBytes <= maximumInt64-fileMultipartOverhead {
			bodyLimit += fileMultipartOverhead
		}
		request.Body = http.MaxBytesReader(response, request.Body, bodyLimit)
		if err := request.ParseMultipartForm(fileMultipartMemory); err != nil {
			var maximumBytesError *http.MaxBytesError
			if errors.As(err, &maximumBytesError) {
				WriteError(response, http.StatusRequestEntityTooLarge, "file_too_large", "File upload is too large", nil)
			} else {
				WriteError(response, http.StatusBadRequest, "invalid_file", "Invalid file upload", nil)
			}
			return
		}
		if request.MultipartForm != nil {
			defer request.MultipartForm.RemoveAll()
		}
		files := request.MultipartForm.File["file"]
		if len(files) != 1 {
			WriteError(response, http.StatusBadRequest, "invalid_file", "Invalid file upload", nil)
			return
		}
		content, err := files[0].Open()
		if err != nil {
			WriteError(response, http.StatusBadRequest, "invalid_file", "Invalid file upload", nil)
			return
		}
		defer content.Close()
		result, err := service.UploadFile(request.Context(), applicationfiles.Upload{
			UploadedBy: user.ID, OriginalName: files[0].Filename,
			ContentType: files[0].Header.Get("Content-Type"), Content: content,
		})
		switch {
		case errors.Is(err, applicationfiles.ErrUploadTooLarge):
			WriteError(response, http.StatusRequestEntityTooLarge, "file_too_large", "File upload is too large", nil)
			return
		case errors.Is(err, applicationfiles.ErrInvalidUpload):
			WriteError(response, http.StatusBadRequest, "invalid_file", "Invalid file upload", nil)
			return
		case err != nil:
			WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			return
		}
		status := http.StatusCreated
		if result.Deduplicated {
			status = http.StatusOK
		}
		writeJSON(response, status, newFileResponse(result))
	})))

	router.Handle(http.MethodGet, fileDownloadPath, RequirePermission(
		authentication,
		domainauth.PermissionFilesRead,
	)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		download, err := service.OpenFile(request.Context(), chi.URLParam(request, "fileID"))
		switch {
		case errors.Is(err, applicationfiles.ErrInvalidFileID), errors.Is(err, domainfiles.ErrFileNotFound):
			WriteError(response, http.StatusNotFound, "file_not_found", "File not found", nil)
			return
		case err != nil:
			WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			return
		}
		defer download.Content.Close()
		disposition := mime.FormatMediaType("attachment", map[string]string{"filename": download.File.OriginalName})
		if disposition == "" {
			disposition = "attachment"
		}
		response.Header().Set("Cache-Control", "private, no-cache")
		response.Header().Set("Content-Type", download.File.ContentType)
		response.Header().Set("Content-Disposition", disposition)
		response.Header().Set("Content-Length", strconv.FormatInt(download.File.SizeBytes, 10))
		response.Header().Set("ETag", `"sha256-`+hex.EncodeToString(download.File.SHA256)+`"`)
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.WriteHeader(http.StatusOK)
		_, _ = io.Copy(response, download.Content)
	})))
}

func newFileResponse(result applicationfiles.UploadResult) fileResponse {
	return fileResponse{
		ID: result.File.ID, SHA256: hex.EncodeToString(result.File.SHA256),
		OriginalName: result.File.OriginalName, ContentType: result.File.ContentType,
		SizeBytes: result.File.SizeBytes, UploadedBy: result.File.UploadedBy,
		CreatedAt: result.File.CreatedAt, Deduplicated: result.Deduplicated,
	}
}
