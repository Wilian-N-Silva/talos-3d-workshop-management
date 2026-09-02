package httpplatform

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

func TestAuthenticationMiddlewareAttachesSafeCurrentUserAndSession(t *testing.T) {
	expiresAt := time.Date(2026, time.September, 2, 15, 0, 0, 0, time.UTC)
	service := &bearerAuthenticationServiceStub{result: applicationauth.AuthenticationResult{
		User: domainauth.User{
			ID:              "user-id",
			Name:            "Workshop Owner",
			EmailOrUsername: "owner@example.com",
			PasswordHash:    "$argon2id$must-not-reach-context",
			Status:          domainauth.UserStatusActive,
			Role:            domainauth.RoleOwner,
		},
		Session: domainauth.Session{
			ID:        "session-id",
			DeviceID:  "device-id",
			TokenHash: []byte("must-not-reach-context"),
			ExpiresAt: expiresAt,
		},
	}}
	called := false
	handler := AuthenticationMiddleware(service)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		called = true
		user, ok := CurrentUserFromContext(request.Context())
		if !ok || user.ID != "user-id" || user.EmailOrUsername != "owner@example.com" || user.Status != domainauth.UserStatusActive || user.Role != domainauth.RoleOwner {
			t.Fatalf("current user = %#v, %t", user, ok)
		}
		session, ok := CurrentSessionFromContext(request.Context())
		if !ok || session.ID != "session-id" || session.DeviceID != "device-id" || !session.ExpiresAt.Equal(expiresAt) {
			t.Fatalf("current session = %#v, %t", session, ok)
		}
		response.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "bEaReR opaque-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent || !called {
		t.Fatalf("response status = %d, handler called = %t", response.Code, called)
	}
	if service.calls != 1 || service.token != "opaque-token" {
		t.Fatalf("authentication service = %d calls with token %q", service.calls, service.token)
	}
}

func TestAuthenticationMiddlewareRejectsMissingAndMalformedHeadersUniformly(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
	}{
		{name: "missing"},
		{name: "wrong scheme", headers: []string{"Basic credentials"}},
		{name: "missing token", headers: []string{"Bearer"}},
		{name: "extra field", headers: []string{"Bearer token extra"}},
		{name: "multiple headers", headers: []string{"Bearer first", "Bearer second"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &bearerAuthenticationServiceStub{}
			handler := AuthenticationMiddleware(service)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("protected handler called")
			}))
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			for _, value := range test.headers {
				request.Header.Add("Authorization", value)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			assertUnauthenticatedResponse(t, response)
			if service.calls != 0 {
				t.Fatalf("authentication service calls = %d, want 0", service.calls)
			}
		})
	}
}

func TestAuthenticationMiddlewareMapsInactiveSessionUniformly(t *testing.T) {
	service := &bearerAuthenticationServiceStub{err: applicationauth.ErrUnauthenticated}
	handler := AuthenticationMiddleware(service)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler called")
	}))
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer expired-or-revoked-token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertUnauthenticatedResponse(t, response)
	if strings.Contains(response.Body.String(), "expired") || strings.Contains(response.Body.String(), "revoked") {
		t.Fatal("unauthenticated response exposed session state")
	}
}

func TestAuthenticationMiddlewareHidesInternalErrors(t *testing.T) {
	service := &bearerAuthenticationServiceStub{err: errors.New("database details")}
	handler := AuthenticationMiddleware(service)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler called")
	}))
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer opaque-token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertAPIError(t, response, http.StatusInternalServerError, "internal_error", "Internal server error")
	if strings.Contains(response.Body.String(), "database details") {
		t.Fatal("authentication response exposed internal error")
	}
}

func TestAuthenticationContextWithoutMiddlewareIsEmpty(t *testing.T) {
	if user, ok := CurrentUserFromContext(context.Background()); ok || user != (CurrentUser{}) {
		t.Fatalf("CurrentUserFromContext() = %#v, %t", user, ok)
	}
	if session, ok := CurrentSessionFromContext(context.Background()); ok || session != (CurrentSession{}) {
		t.Fatalf("CurrentSessionFromContext() = %#v, %t", session, ok)
	}
}

func assertUnauthenticatedResponse(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	assertAPIError(t, response, http.StatusUnauthorized, "unauthenticated", "Authentication required")
	if got := response.Header().Get("WWW-Authenticate"); got != "Bearer" {
		t.Fatalf("WWW-Authenticate = %q, want Bearer", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

type bearerAuthenticationServiceStub struct {
	result applicationauth.AuthenticationResult
	err    error
	token  string
	calls  int
}

func (stub *bearerAuthenticationServiceStub) Authenticate(
	_ context.Context,
	token string,
) (applicationauth.AuthenticationResult, error) {
	stub.calls++
	stub.token = token
	return stub.result, stub.err
}
