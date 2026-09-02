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
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
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

func (stub readinessStub) Check(context.Context) error {
	return stub.err
}

type setupServiceStub struct {
	needsSetup bool
}

type loginServiceStub struct{}

type authenticationServiceStub struct{}

type sessionManagementServiceStub struct{}

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

func TestHandlerHasNoProductRoutes(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/catalog", nil)
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
	)
}
