package httpplatform

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	applicationsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/settings"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
)

func TestWorkshopLogoUploadRequiresSettingsManageAndAssociatesUploader(t *testing.T) {
	content := httpTestPNG(t)
	body, contentType := multipartLogo(t, "workshop.png", content)

	t.Run("owner", func(t *testing.T) {
		service := &workshopLogoServiceStub{uploadResult: applicationsettings.LogoUploadResult{File: domainfiles.File{ID: "file-id"}}}
		authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleOwner)
		response := serveWorkshopLogoUpload(t, body, contentType, authentication, service)
		if response.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201: %s", response.Code, response.Body.String())
		}
		if service.uploadCalls != 1 || service.upload.UploadedBy != httpTestUserID || service.upload.OriginalName != "workshop.png" {
			t.Fatalf("upload = %#v, calls = %d", service.upload, service.uploadCalls)
		}
		gotContent, _ := io.ReadAll(service.upload.Content)
		if !bytes.Equal(gotContent, content) {
			t.Fatal("service did not receive uploaded image bytes")
		}
		var got workshopLogoResponse
		if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if got.FileID != "file-id" || got.LogoURL != APIV1Prefix+WorkshopLogoDownloadPath {
			t.Fatalf("upload response = %#v", got)
		}
	})

	t.Run("viewer", func(t *testing.T) {
		service := &workshopLogoServiceStub{}
		authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleViewer)
		response := serveWorkshopLogoUpload(t, body, contentType, authentication, service)
		assertAPIError(t, response, http.StatusForbidden, "forbidden", "Permission denied")
		if service.uploadCalls != 0 {
			t.Fatalf("upload calls = %d, want 0", service.uploadCalls)
		}
	})
}

func TestWorkshopLogoUploadMapsInvalidImage(t *testing.T) {
	body, contentType := multipartLogo(t, "workshop.png", []byte("not an image"))
	service := &workshopLogoServiceStub{uploadError: applicationsettings.ErrInvalidWorkshopLogo}
	authentication := authenticatedSessionStub(httpTestUserID, httpTestSessionID, domainauth.RoleOwner)
	response := serveWorkshopLogoUpload(t, body, contentType, authentication, service)
	assertAPIError(t, response, http.StatusBadRequest, "invalid_logo", "Invalid workshop logo")
}

func TestWorkshopLogoDownloadReturnsOnlyCurrentPublicBranding(t *testing.T) {
	content := httpTestPNG(t)
	digest := sha256.Sum256(content)
	service := &workshopLogoServiceStub{download: applicationsettings.LogoDownload{
		File: domainfiles.File{
			ID:          "current-file",
			SHA256:      digest[:],
			ContentType: "image/png",
			SizeBytes:   int64(len(content)),
		},
		Content: io.NopCloser(bytes.NewReader(content)),
	}}
	router := NewAPIV1Router()
	RegisterWorkshopLogo(router, &bearerAuthenticationServiceStub{}, service, applicationsettings.DefaultMaximumLogoBytes)
	mux := http.NewServeMux()
	RegisterAPIV1(mux, router)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, APIV1Prefix+WorkshopLogoDownloadPath, nil))

	if response.Code != http.StatusOK || !bytes.Equal(response.Body.Bytes(), content) {
		t.Fatalf("download = %d, %d bytes", response.Code, response.Body.Len())
	}
	if response.Header().Get("Content-Type") != "image/png" || response.Header().Get("X-Content-Type-Options") != "nosniff" ||
		response.Header().Get("ETag") == "" || service.openCalls != 1 {
		t.Fatalf("download headers = %#v, calls = %d", response.Header(), service.openCalls)
	}
}

func TestWorkshopLogoDownloadMapsMissingLogo(t *testing.T) {
	service := &workshopLogoServiceStub{openError: domainfiles.ErrFileNotFound}
	router := NewAPIV1Router()
	RegisterWorkshopLogo(router, &bearerAuthenticationServiceStub{}, service, applicationsettings.DefaultMaximumLogoBytes)
	response := serveAPIRequest(t, router, http.MethodGet, APIV1Prefix+WorkshopLogoDownloadPath, "")
	assertAPIError(t, response, http.StatusNotFound, "logo_not_found", "Workshop logo not found")
}

func serveWorkshopLogoUpload(
	t *testing.T,
	body []byte,
	contentType string,
	authentication BearerAuthenticationService,
	service WorkshopLogoService,
) *httptest.ResponseRecorder {
	t.Helper()
	router := NewAPIV1Router()
	RegisterWorkshopLogo(router, authentication, service, applicationsettings.DefaultMaximumLogoBytes)
	mux := http.NewServeMux()
	RegisterAPIV1(mux, router)
	request := httptest.NewRequest(http.MethodPost, APIV1Prefix+WorkshopLogoUploadPath, bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func multipartLogo(t *testing.T, name string, content []byte) ([]byte, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart body: %v", err)
	}
	return body.Bytes(), writer.FormDataContentType()
}

func httpTestPNG(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return buffer.Bytes()
}

type workshopLogoServiceStub struct {
	upload       applicationsettings.LogoUpload
	uploadResult applicationsettings.LogoUploadResult
	uploadError  error
	uploadCalls  int
	download     applicationsettings.LogoDownload
	openError    error
	openCalls    int
}

func (stub *workshopLogoServiceStub) Upload(
	_ context.Context,
	upload applicationsettings.LogoUpload,
) (applicationsettings.LogoUploadResult, error) {
	stub.uploadCalls++
	content, _ := io.ReadAll(upload.Content)
	upload.Content = bytes.NewReader(content)
	stub.upload = upload
	return stub.uploadResult, stub.uploadError
}

func (stub *workshopLogoServiceStub) OpenCurrent(context.Context) (applicationsettings.LogoDownload, error) {
	stub.openCalls++
	return stub.download, stub.openError
}
