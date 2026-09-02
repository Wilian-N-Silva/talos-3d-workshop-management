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

func TestLoginReturnsOpaqueSessionAndSafeMetadata(t *testing.T) {
	loggedInAt := time.Date(2026, time.September, 1, 15, 0, 0, 0, time.UTC)
	expiresAt := loggedInAt.Add(24 * time.Hour)
	service := &loginServiceStub{result: applicationauth.LoginResult{
		User: domainauth.User{
			ID:              "user-id",
			Name:            "Owner",
			EmailOrUsername: "owner@example.com",
			PasswordHash:    "$argon2id$must-not-be-returned",
			Status:          domainauth.UserStatusActive,
			LastLoginAt:     &loggedInAt,
		},
		Device: domainauth.ClientDevice{
			ID:          "device-id",
			DisplayName: "Workshop PC",
			OS:          "Windows 11",
			AppVersion:  "1.0.0",
			LastSeenAt:  loggedInAt,
		},
		Session: applicationauth.IssuedSession{
			Token: "opaque-session-token",
			Session: domainauth.Session{
				TokenHash: []byte("must-not-be-returned"),
				ExpiresAt: expiresAt,
			},
		},
	}}
	body := `{"email_or_username":"owner@example.com","password":"correct horse battery staple","device":{"display_name":"Workshop PC","os":"Windows 11","app_version":"1.0.0"}}`
	response := serveLoginRequest(t, service, permissiveLoginLimiter(t), body, "127.0.0.1:1234")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	if service.calls != 1 || service.input.Password == "" || service.input.Device.DisplayName != "Workshop PC" {
		t.Fatalf("login service input = %#v with %d calls", service.input, service.calls)
	}
	if strings.Contains(response.Body.String(), "argon2id") || strings.Contains(response.Body.String(), "must-not-be-returned") || strings.Contains(response.Body.String(), "correct horse") {
		t.Fatal("login response exposed password or hash material")
	}
	var got loginResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if got.Token != "opaque-session-token" || !got.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("session response = token %q, expiry %s", got.Token, got.ExpiresAt)
	}
	if got.User.ID != "user-id" || got.Device.ID != "device-id" {
		t.Fatalf("login metadata = %#v", got)
	}
}

func TestLoginMapsInvalidCredentialsUniformly(t *testing.T) {
	service := &loginServiceStub{err: applicationauth.ErrInvalidCredentials}
	body := `{"email_or_username":"unknown","password":"wrong credentials","device":{"display_name":"PC","os":"Windows","app_version":"dev"}}`
	response := serveLoginRequest(t, service, permissiveLoginLimiter(t), body, "127.0.0.1:1234")

	assertAPIError(t, response, http.StatusUnauthorized, "invalid_credentials", "Invalid credentials")
	if strings.Contains(response.Body.String(), "unknown") || strings.Contains(response.Body.String(), "password") {
		t.Fatal("invalid credential response exposed field or account details")
	}
}

func TestLoginMapsInvalidDeviceAndInternalErrors(t *testing.T) {
	body := `{"email_or_username":"owner","password":"a long owner passphrase","device":{"display_name":"PC","os":"Windows","app_version":"dev"}}`
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid device", err: applicationauth.ErrInvalidLoginDevice, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "internal", err: errors.New("database details"), wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := serveLoginRequest(t, &loginServiceStub{err: test.err}, permissiveLoginLimiter(t), body, "127.0.0.1:1234")
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			var envelope ErrorEnvelope
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if envelope.Error.Code != test.wantCode || strings.Contains(response.Body.String(), "database details") {
				t.Fatalf("error response = %s", response.Body.String())
			}
		})
	}
}

func TestLoginRejectsMalformedRequest(t *testing.T) {
	service := &loginServiceStub{}
	response := serveLoginRequest(t, service, permissiveLoginLimiter(t), `{"unknown":true}`, "127.0.0.1:1234")

	assertAPIError(t, response, http.StatusBadRequest, "invalid_request", "Invalid request")
	if service.calls != 0 {
		t.Fatalf("service calls = %d, want 0", service.calls)
	}
}

func TestLoginRateLimitsBySocketPeerAndIgnoresForwardedHeader(t *testing.T) {
	service := &loginServiceStub{err: applicationauth.ErrInvalidCredentials}
	limiter, err := NewLoginRateLimiter(2, time.Minute)
	if err != nil {
		t.Fatalf("NewLoginRateLimiter() error = %v", err)
	}
	body := `{"email_or_username":"owner","password":"a long owner passphrase","device":{"display_name":"PC","os":"Windows","app_version":"dev"}}`
	router := NewAPIV1Router()
	RegisterLogin(router, service, limiter)

	for attempt := 1; attempt <= 3; attempt++ {
		mux := http.NewServeMux()
		RegisterAPIV1(mux, router)
		request := httptest.NewRequest(http.MethodPost, APIV1Prefix+LoginPath, strings.NewReader(body))
		request.RemoteAddr = "192.0.2.1:1234"
		request.Header.Set("X-Forwarded-For", "198.51.100."+string(rune('0'+attempt)))
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)

		if attempt <= 2 && response.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", attempt, response.Code)
		}
		if attempt == 3 {
			assertAPIError(t, response, http.StatusTooManyRequests, "rate_limited", "Too many login attempts")
			if response.Header().Get("Retry-After") == "" {
				t.Fatal("rate-limited response has no Retry-After")
			}
		}
	}
	if service.calls != 2 {
		t.Fatalf("service calls = %d, want 2", service.calls)
	}
}

func serveLoginRequest(
	t *testing.T,
	service LoginService,
	limiter *LoginRateLimiter,
	body string,
	remoteAddress string,
) *httptest.ResponseRecorder {
	t.Helper()
	router := NewAPIV1Router()
	RegisterLogin(router, service, limiter)
	mux := http.NewServeMux()
	RegisterAPIV1(mux, router)
	request := httptest.NewRequest(http.MethodPost, APIV1Prefix+LoginPath, strings.NewReader(body))
	request.RemoteAddr = remoteAddress
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}

func permissiveLoginLimiter(t *testing.T) *LoginRateLimiter {
	t.Helper()
	limiter, err := NewLoginRateLimiter(100, time.Minute)
	if err != nil {
		t.Fatalf("NewLoginRateLimiter() error = %v", err)
	}
	return limiter
}

type loginServiceStub struct {
	result applicationauth.LoginResult
	err    error
	input  applicationauth.LoginInput
	calls  int
}

func (stub *loginServiceStub) Login(_ context.Context, input applicationauth.LoginInput) (applicationauth.LoginResult, error) {
	stub.calls++
	stub.input = input
	return stub.result, stub.err
}
