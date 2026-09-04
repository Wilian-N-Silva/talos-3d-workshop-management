package httpplatform

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	applicationfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/files"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
)

func TestFileUploadRequiresPermissionAndReturnsCanonicalMetadata(t *testing.T) {
	createdAt := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	service := &fileTransferServiceStub{uploadResult: applicationfiles.UploadResult{File: domainfiles.File{
		ID: "file-id", SHA256: bytes.Repeat([]byte{0xab}, 32), OriginalName: "part.stl",
		ContentType: "model/stl", SizeBytes: 4, UploadedBy: "designer-id", CreatedAt: createdAt,
	}}}
	authentication := authorizedFileUser(domainauth.RoleDesigner)
	body, contentType := multipartFile(t, "part.stl", "mesh")
	request := httptest.NewRequest(http.MethodPost, FilesPath, body)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	router := NewAPIV1Router()
	RegisterFiles(router, authentication, service, 100)
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || service.upload.UploadedBy != "designer-id" || service.upload.OriginalName != "part.stl" {
		t.Fatalf("upload status = %d, input = %#v, body = %s", response.Code, service.upload, response.Body.String())
	}
	var got fileResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil || got.ID != "file-id" || got.SHA256 != strings.Repeat("ab", 32) || got.Deduplicated {
		t.Fatalf("upload response = %#v, %v", got, err)
	}

	forbiddenBody, forbiddenType := multipartFile(t, "part.stl", "mesh")
	forbiddenRequest := httptest.NewRequest(http.MethodPost, FilesPath, forbiddenBody)
	forbiddenRequest.Header.Set("Content-Type", forbiddenType)
	forbiddenRequest.Header.Set("Authorization", "Bearer token")
	forbidden := httptest.NewRecorder()
	forbiddenRouter := NewAPIV1Router()
	RegisterFiles(forbiddenRouter, authorizedFileUser(domainauth.RoleViewer), service, 100)
	forbiddenRouter.ServeHTTP(forbidden, forbiddenRequest)
	assertAPIError(t, forbidden, http.StatusForbidden, "forbidden", "Permission denied")
}

func TestFileDownloadStreamsSafeAttachmentAndMapsNotFound(t *testing.T) {
	service := &fileTransferServiceStub{download: applicationfiles.Download{File: domainfiles.File{
		ID: "11111111-1111-4111-8111-111111111111", SHA256: bytes.Repeat([]byte{0xcd}, 32),
		OriginalName: `model "final".3mf`, ContentType: "application/zip", SizeBytes: 7,
	}, Content: io.NopCloser(strings.NewReader("content"))}}
	router := NewAPIV1Router()
	RegisterFiles(router, authorizedFileUser(domainauth.RoleViewer), service, 100)
	request := httptest.NewRequest(http.MethodGet, FilesPath+"/11111111-1111-4111-8111-111111111111", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "content" || response.Header().Get("Content-Disposition") != `attachment; filename="model \"final\".3mf"` || response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("download status/headers/body = %d, %#v, %q", response.Code, response.Header(), response.Body.String())
	}

	service.download = applicationfiles.Download{}
	service.downloadError = domainfiles.ErrFileNotFound
	notFound := httptest.NewRecorder()
	router.ServeHTTP(notFound, request)
	assertAPIError(t, notFound, http.StatusNotFound, "file_not_found", "File not found")
}

func TestFileDownloadForbidsRoleWithoutReadPermission(t *testing.T) {
	router := NewAPIV1Router()
	service := &fileTransferServiceStub{}
	RegisterFiles(router, authorizedFileUser(domainauth.RoleCommercial), service, 100)
	request := httptest.NewRequest(http.MethodGet, FilesPath+"/11111111-1111-4111-8111-111111111111", nil)
	request.Header.Set("Authorization", "Bearer token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	assertAPIError(t, response, http.StatusForbidden, "forbidden", "Permission denied")
	if service.openID != "" {
		t.Fatal("download service called despite forbidden permission")
	}
}

func TestFileUploadMapsMalformedAndOversizedRequests(t *testing.T) {
	router := NewAPIV1Router()
	service := &fileTransferServiceStub{}
	RegisterFiles(router, authorizedFileUser(domainauth.RoleDesigner), service, 100)

	malformedRequest := httptest.NewRequest(http.MethodPost, FilesPath, strings.NewReader("not multipart"))
	malformedRequest.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	malformedRequest.Header.Set("Authorization", "Bearer token")
	malformed := httptest.NewRecorder()
	router.ServeHTTP(malformed, malformedRequest)
	assertAPIError(t, malformed, http.StatusBadRequest, "invalid_file", "Invalid file upload")

	service.uploadError = applicationfiles.ErrUploadTooLarge
	body, contentType := multipartFile(t, "large.bin", "content")
	oversizedRequest := httptest.NewRequest(http.MethodPost, FilesPath, body)
	oversizedRequest.Header.Set("Content-Type", contentType)
	oversizedRequest.Header.Set("Authorization", "Bearer token")
	oversized := httptest.NewRecorder()
	router.ServeHTTP(oversized, oversizedRequest)
	assertAPIError(t, oversized, http.StatusRequestEntityTooLarge, "file_too_large", "File upload is too large")
}

func multipartFile(t *testing.T, filename, content string) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	_, _ = io.WriteString(part, content)
	_ = writer.Close()
	return &body, writer.FormDataContentType()
}

func authorizedFileUser(role domainauth.Role) *bearerAuthenticationServiceStub {
	return &bearerAuthenticationServiceStub{result: applicationauth.AuthenticationResult{
		User: domainauth.User{ID: "designer-id", Status: domainauth.UserStatusActive, Role: role},
	}}
}

type fileTransferServiceStub struct {
	upload        applicationfiles.Upload
	uploadResult  applicationfiles.UploadResult
	uploadError   error
	download      applicationfiles.Download
	downloadError error
	openID        string
}

func (stub *fileTransferServiceStub) UploadFile(_ context.Context, upload applicationfiles.Upload) (applicationfiles.UploadResult, error) {
	stub.upload = upload
	return stub.uploadResult, stub.uploadError
}

func (stub *fileTransferServiceStub) OpenFile(_ context.Context, id string) (applicationfiles.Download, error) {
	stub.openID = id
	if stub.downloadError != nil {
		return applicationfiles.Download{}, stub.downloadError
	}
	return stub.download, nil
}
