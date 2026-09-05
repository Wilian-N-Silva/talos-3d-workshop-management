package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	applicationcatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/catalog"
	applicationfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/files"
	applicationsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/settings"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
	httpplatform "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/platform/http"
)

type readinessStub struct {
	err error
}

var testMetadata = httpplatform.MetaResponse{
	APIVersion:            httpplatform.APIVersion,
	ServerVersion:         "test",
	WorkshopName:          "Test Workshop",
	MinimumDesktopVersion: "0.0.0",
}

var testSetupService = setupServiceStub{needsSetup: true}
var testLoginService = loginServiceStub{}
var testAuthenticationService = authenticationServiceStub{}
var testSessionManagementService = sessionManagementServiceStub{}
var testWorkshopSettingsService = workshopSettingsServiceStub{}
var testWorkshopLogoService = workshopLogoServiceStub{}
var testFileTransferService = fileTransferServiceStub{}
var testCatalogItemService = catalogItemServiceStub{}
var testCatalogDesignService = catalogDesignServiceStub{}

func (stub readinessStub) Check(context.Context) error {
	return stub.err
}

type setupServiceStub struct {
	needsSetup bool
}

type loginServiceStub struct{}

type authenticationServiceStub struct{}

type sessionManagementServiceStub struct{}

type workshopSettingsServiceStub struct{}

type workshopLogoServiceStub struct{}
type fileTransferServiceStub struct{}
type catalogItemServiceStub struct{}
type catalogDesignServiceStub struct{}

func (loginServiceStub) Login(
	context.Context,
	applicationauth.LoginInput,
) (applicationauth.LoginResult, error) {
	return applicationauth.LoginResult{}, nil
}

func (authenticationServiceStub) Authenticate(
	context.Context,
	string,
) (applicationauth.AuthenticationResult, error) {
	return applicationauth.AuthenticationResult{
		User: domainauth.User{
			ID:     "11111111-1111-4111-8111-111111111111",
			Status: domainauth.UserStatusActive,
			Role:   domainauth.RoleViewer,
		},
		Session: domainauth.Session{
			ID:        "33333333-3333-4333-8333-333333333333",
			DeviceID:  "44444444-4444-4444-8444-444444444444",
			ExpiresAt: time.Now().Add(time.Hour),
		},
	}, nil
}

func (sessionManagementServiceStub) List(
	context.Context,
	applicationauth.SessionActor,
	string,
) ([]domainauth.SessionDetails, error) {
	return []domainauth.SessionDetails{}, nil
}

func (sessionManagementServiceStub) Revoke(
	context.Context,
	applicationauth.SessionActor,
	string,
) (domainauth.Session, error) {
	return domainauth.Session{}, nil
}

func (workshopSettingsServiceStub) Get(context.Context) (domainsettings.WorkshopSettings, error) {
	return domainsettings.WorkshopSettings{
		WorkshopName:    testMetadata.WorkshopName,
		DefaultLocale:   "pt-BR",
		DefaultCurrency: "BRL",
		DisplayTimezone: "America/Sao_Paulo",
		DefaultTheme:    domainsettings.ThemeSystem,
	}, nil
}

func (workshopSettingsServiceStub) Update(
	context.Context,
	domainsettings.Values,
) (domainsettings.WorkshopSettings, error) {
	return domainsettings.WorkshopSettings{}, nil
}

func (workshopLogoServiceStub) Upload(
	context.Context,
	applicationsettings.LogoUpload,
) (applicationsettings.LogoUploadResult, error) {
	return applicationsettings.LogoUploadResult{}, nil
}

func (workshopLogoServiceStub) OpenCurrent(context.Context) (applicationsettings.LogoDownload, error) {
	return applicationsettings.LogoDownload{}, domainfiles.ErrFileNotFound
}

func (stub setupServiceStub) NeedsSetup(context.Context) (bool, error) {
	return stub.needsSetup, nil
}

func (setupServiceStub) CreateAdmin(
	context.Context,
	applicationauth.CreateAdminInput,
) (domainauth.User, error) {
	return domainauth.User{}, nil
}

func TestHandlerRegistersLiveness(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{err: errors.New("dependencies unavailable")}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestHandlerRegistersReadiness(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestHandlerRegistersMeta(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.MetaPath, nil)
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var got httpplatform.MetaResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if got != testMetadata {
		t.Fatalf("metadata = %#v, want %#v", got, testMetadata)
	}
}

func TestHandlerRegistersSetupStatus(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.SetupPrefix+httpplatform.SetupStatusPath, nil)
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var got struct {
		NeedsSetup bool `json:"needs_setup"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode setup status: %v", err)
	}
	if !got.NeedsSetup {
		t.Fatal("needs_setup = false, want true")
	}
}

func TestHandlerRejectsUnknownProductRoutes(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/unknown", nil)
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotFound)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID is empty")
	}
}

func TestRunStopsWhenContextIsCancelled(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	handler := newTestHandler(t, readinessStub{})
	go func() {
		result <- run(ctx, listener, log.New(io.Discard, "", 0), handler)
	}()

	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + listener.Addr().String() + "/health/live")
	if err != nil {
		cancel()
		t.Fatalf("request liveness: %v", err)
	}
	_ = response.Body.Close()

	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after context cancellation")
	}
}

func TestHandlerRegistersLogin(t *testing.T) {
	body := `{"email_or_username":"owner","password":"a long owner passphrase","device":{"display_name":"PC","os":"Windows","app_version":"dev"}}`
	request := httptest.NewRequest(http.MethodPost, httpplatform.APIV1Prefix+httpplatform.LoginPath, strings.NewReader(body))
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestHandlerRegistersSessionManagement(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.SessionsPath, nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHandlerRegistersWorkshopSettings(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.WorkshopSettingsPath, nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestHandlerRegistersWorkshopLogo(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.WorkshopLogoDownloadPath, nil)
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusNotFound, response.Body.String())
	}
}

func (fileTransferServiceStub) UploadFile(context.Context, applicationfiles.Upload) (applicationfiles.UploadResult, error) {
	return applicationfiles.UploadResult{}, nil
}

func (fileTransferServiceStub) OpenFile(context.Context, string) (applicationfiles.Download, error) {
	return applicationfiles.Download{}, domainfiles.ErrFileNotFound
}

func (catalogItemServiceStub) Create(context.Context, domaincatalog.Values) (domaincatalog.Item, error) {
	return domaincatalog.Item{}, nil
}

func (catalogItemServiceStub) Get(context.Context, string) (domaincatalog.Item, error) {
	return domaincatalog.Item{}, domaincatalog.ErrItemNotFound
}

func (catalogItemServiceStub) List(_ context.Context, filter domaincatalog.ListFilter) (domaincatalog.Page, error) {
	if filter.Limit == 0 {
		filter.Limit = applicationcatalog.DefaultListLimit
	}
	return domaincatalog.Page{Items: []domaincatalog.Item{}, Limit: filter.Limit, Offset: filter.Offset}, nil
}

func (catalogItemServiceStub) Update(context.Context, string, domaincatalog.Values) (domaincatalog.Item, error) {
	return domaincatalog.Item{}, nil
}

func (catalogItemServiceStub) Delete(context.Context, string) error {
	return nil
}

func (catalogDesignServiceStub) CreatePart(context.Context, string, domaincatalog.PartValues) (domaincatalog.Part, error) {
	return domaincatalog.Part{}, nil
}
func (catalogDesignServiceStub) GetPart(context.Context, string) (domaincatalog.Part, error) {
	return domaincatalog.Part{}, domaincatalog.ErrPartNotFound
}
func (catalogDesignServiceStub) ListParts(context.Context, string) ([]domaincatalog.Part, error) {
	return []domaincatalog.Part{}, nil
}
func (catalogDesignServiceStub) UpdatePart(context.Context, string, domaincatalog.PartValues) (domaincatalog.Part, error) {
	return domaincatalog.Part{}, nil
}
func (catalogDesignServiceStub) DeletePart(context.Context, string) error { return nil }
func (catalogDesignServiceStub) CreateVersion(context.Context, string, string, domaincatalog.DesignVersionValues) (domaincatalog.DesignVersion, error) {
	return domaincatalog.DesignVersion{}, nil
}
func (catalogDesignServiceStub) ListVersions(context.Context, string) ([]domaincatalog.DesignVersion, error) {
	return []domaincatalog.DesignVersion{}, nil
}
func (catalogDesignServiceStub) AttachFile(context.Context, string, string, string, domaincatalog.DesignFileRole) (domaincatalog.DesignFile, error) {
	return domaincatalog.DesignFile{}, nil
}

func TestHandlerRegistersFiles(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.FilesPath+"/11111111-1111-4111-8111-111111111111", nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusNotFound, response.Body.String())
	}
}

func TestHandlerRegistersCatalogItems(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, httpplatform.APIV1Prefix+httpplatform.CatalogItemsPath, nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()

	newTestHandler(t, readinessStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func newTestHandler(t *testing.T, readiness httpplatform.ReadinessChecker) http.Handler {
	t.Helper()
	limiter, err := httpplatform.NewLoginRateLimiter(100, time.Minute)
	if err != nil {
		t.Fatalf("NewLoginRateLimiter() error = %v", err)
	}
	return newHandler(
		readiness,
		testMetadata,
		testSetupService,
		testLoginService,
		limiter,
		testAuthenticationService,
		testSessionManagementService,
		testWorkshopSettingsService,
		testWorkshopLogoService,
		applicationsettings.DefaultMaximumLogoBytes,
		testFileTransferService,
		1024,
		testCatalogItemService,
		testCatalogDesignService,
	)
}
