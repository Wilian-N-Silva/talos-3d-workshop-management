package httpplatform

import (
	"net/http"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

// AuthorizationMiddleware requires one server permission from an already
// authenticated request. It remains deny-by-default when authentication
// context is absent or a role is unknown.
func AuthorizationMiddleware(permission domainauth.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			user, ok := CurrentUserFromContext(request.Context())
			if !ok {
				writeUnauthenticated(response)
				return
			}
			if !user.HasPermission(permission) {
				response.Header().Set("Cache-Control", "no-store")
				WriteError(response, http.StatusForbidden, "forbidden", "Permission denied", nil)
				return
			}
			next.ServeHTTP(response, request)
		})
	}
}

// RequirePermission composes bearer authentication and authorization in the
// correct order for protected handlers.
func RequirePermission(
	service BearerAuthenticationService,
	permission domainauth.Permission,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return AuthenticationMiddleware(service)(AuthorizationMiddleware(permission)(next))
	}
}
