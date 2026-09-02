package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

func TestSetupStatusReturnsAvailability(t *testing.T) {
	service := &setupServiceStub{needsSetup: true}
	response := serveSetupRequest(t, service, http.MethodGet, SetupPrefix+SetupStatusPath, "", "setup-status")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
	if got := response.Header().Get(RequestIDHeader); got != "setup-status" {
		t.Fatalf("X-Request-ID = %q, want setup-status", got)
	}
	var body setupStatusResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode setup status: %v", err)
	}
	if !body.NeedsSetup || service.statusCalls != 1 {
		t.Fatalf("setup status = %#v with %d calls", body, service.statusCalls)
	}
}

func TestSetupStatusHidesInternalErrors(t *testing.T) {
	service := &setupServiceStub{statusError: errors.New("database details")}
	response := serveSetupRequest(t, service, http.MethodGet, SetupPrefix+SetupStatusPath, "", "")

	assertAPIError(t, response, http.StatusInternalServerError, "internal_error", "Internal server error")
	if strings.Contains(response.Body.String(), "database details") {
		t.Fatal("setup status exposed internal error details")
	}
}

func TestSetupAdminCreatesOwnerWithoutExposingPasswordHash(t *testing.T) {
	createdAt := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	service := &setupServiceStub{createdUser: domainauth.User{
		ID:              "owner-id",
		Name:            "Workshop Owner",
		EmailOrUsername: "owner@example.com",
		PasswordHash:    "$argon2id$must-not-be-returned",
		Status:          domainauth.UserStatusActive,
		Role:            domainauth.RoleOwner,
		CreatedAt:       createdAt,
	}}
	body := `{"name":"Workshop Owner","email_or_username":"owner@example.com","password":"a long owner passphrase"}`
	response := serveSetupRequest(t, service, http.MethodPost, SetupPrefix+SetupAdminPath, body, "setup-create")

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	if service.createCalls != 1 || !service.receivedPassword {
		t.Fatalf("create calls = %d, received password = %t", service.createCalls, service.receivedPassword)
	}
	if strings.Contains(response.Body.String(), "argon2id") || strings.Contains(response.Body.String(), "passphrase") {
		t.Fatal("setup response exposed password material")
	}

	var responseBody createdAdminResponse
	if err := json.Unmarshal(response.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode created owner: %v", err)
	}
	if responseBody.User.ID != "owner-id" || responseBody.User.Status != domainauth.UserStatusActive || responseBody.User.Role != domainauth.RoleOwner || len(responseBody.User.Permissions) != len(domainauth.AllPermissions()) {
		t.Fatalf("created owner response = %#v", responseBody)
	}
}

func TestSetupAdminMapsApplicationErrors(t *testing.T) {
	tests := []struct {
		name        string
		serviceErr  error
		wantStatus  int
		wantCode    string
		wantDetails map[string]any
	}{
		{name: "closed", serviceErr: applicationauth.ErrSetupClosed, wantStatus: http.StatusConflict, wantCode: "setup_closed"},
		{name: "invalid name", serviceErr: applicationauth.ErrInvalidName, wantStatus: http.StatusBadRequest, wantCode: "invalid_request", wantDetails: map[string]any{"field": "name"}},
		{name: "invalid login", serviceErr: applicationauth.ErrInvalidLogin, wantStatus: http.StatusBadRequest, wantCode: "invalid_request", wantDetails: map[string]any{"field": "email_or_username"}},
		{name: "invalid password", serviceErr: applicationauth.ErrInvalidPassword, wantStatus: http.StatusBadRequest, wantCode: "invalid_request", wantDetails: map[string]any{"field": "password"}},
		{name: "internal", serviceErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &setupServiceStub{createError: test.serviceErr}
			body := `{"name":"Owner","email_or_username":"owner","password":"a long owner passphrase"}`
			response := serveSetupRequest(t, service, http.MethodPost, SetupPrefix+SetupAdminPath, body, "")

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			var envelope ErrorEnvelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if envelope.Error.Code != test.wantCode {
				t.Fatalf("error code = %q, want %q", envelope.Error.Code, test.wantCode)
			}
			for key, want := range test.wantDetails {
				if got := envelope.Error.Details[key]; got != want {
					t.Fatalf("error detail %q = %#v, want %#v", key, got, want)
				}
			}
		})
	}
}

func TestSetupAdminRejectsMalformedRequest(t *testing.T) {
	service := &setupServiceStub{}
	response := serveSetupRequest(
		t,
		service,
		http.MethodPost,
		SetupPrefix+SetupAdminPath,
		`{"name":"Owner","unknown":true}`,
		"",
	)

	assertAPIError(t, response, http.StatusBadRequest, "invalid_request", "Invalid request")
	if service.createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", service.createCalls)
	}
}

func TestSetupRoutesRejectUnsupportedMethodsWithJSON(t *testing.T) {
	response := serveSetupRequest(t, &setupServiceStub{}, http.MethodPost, SetupPrefix+SetupStatusPath, "", "")

	assertAPIError(t, response, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	if got := response.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("Allow = %q, want GET", got)
	}
}

func serveSetupRequest(
	t *testing.T,
	service SetupService,
	method string,
	path string,
	body string,
	requestID string,
) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	registerSetup(mux, service, func() (string, error) { return "generated-setup-id", nil })
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if requestID != "" {
		request.Header.Set(RequestIDHeader, requestID)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

type setupServiceStub struct {
	needsSetup       bool
	statusError      error
	statusCalls      int
	createdUser      domainauth.User
	createError      error
	createCalls      int
	receivedPassword bool
}

func (stub *setupServiceStub) NeedsSetup(context.Context) (bool, error) {
	stub.statusCalls++
	return stub.needsSetup, stub.statusError
}

func (stub *setupServiceStub) CreateAdmin(
	_ context.Context,
	input applicationauth.CreateAdminInput,
) (domainauth.User, error) {
	stub.createCalls++
	stub.receivedPassword = input.Password != ""
	return stub.createdUser, stub.createError
}
