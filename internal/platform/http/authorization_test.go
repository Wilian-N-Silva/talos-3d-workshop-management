package httpplatform

import (
	"net/http"
	"net/http/httptest"
	"testing"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

func TestRequirePermissionAllowsAuthorizedAuthenticatedUser(t *testing.T) {
	service := &bearerAuthenticationServiceStub{result: applicationauth.AuthenticationResult{
		User:    domainauth.User{ID: "designer-id", Status: domainauth.UserStatusActive, Role: domainauth.RoleDesigner},
		Session: domainauth.Session{ID: "session-id"},
	}}
	called := false
	handler := RequirePermission(service, domainauth.PermissionFilesUpload)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		called = true
		user, ok := CurrentUserFromContext(request.Context())
		if !ok || user.Role != domainauth.RoleDesigner || !user.HasPermission(domainauth.PermissionFilesUpload) {
			t.Fatalf("authorized current user = %#v, %t", user, ok)
		}
		response.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodPost, "/protected", nil)
	request.Header.Set("Authorization", "Bearer opaque-token")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent || !called {
		t.Fatalf("status = %d, handler called = %t", response.Code, called)
	}
}

func TestRequirePermissionDistinguishesUnauthenticatedAndForbidden(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		handler := RequirePermission(&bearerAuthenticationServiceStub{}, domainauth.PermissionCatalogRead)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("protected handler called")
		}))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/protected", nil))
		assertUnauthenticatedResponse(t, response)
	})

	t.Run("forbidden", func(t *testing.T) {
		service := &bearerAuthenticationServiceStub{result: applicationauth.AuthenticationResult{
			User: domainauth.User{ID: "viewer-id", Status: domainauth.UserStatusActive, Role: domainauth.RoleViewer},
		}}
		handler := RequirePermission(service, domainauth.PermissionUsersManage)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatal("protected handler called")
		}))
		request := httptest.NewRequest(http.MethodGet, "/protected", nil)
		request.Header.Set("Authorization", "Bearer opaque-token")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		assertAPIError(t, response, http.StatusForbidden, "forbidden", "Permission denied")
		if got := response.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("Cache-Control = %q, want no-store", got)
		}
	})
}

func TestAuthorizationMiddlewareDeniesMissingAuthenticationContext(t *testing.T) {
	handler := AuthorizationMiddleware(domainauth.PermissionCatalogRead)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler called")
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/protected", nil))
	assertUnauthenticatedResponse(t, response)
}
