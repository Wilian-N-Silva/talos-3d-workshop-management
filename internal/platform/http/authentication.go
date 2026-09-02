package httpplatform

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

// BearerAuthenticationService resolves an opaque session token.
type BearerAuthenticationService interface {
	Authenticate(context.Context, string) (applicationauth.AuthenticationResult, error)
}

// CurrentUser is the password-hash-free identity made available to handlers.
type CurrentUser struct {
	ID              string
	Name            string
	EmailOrUsername string
	Status          domainauth.UserStatus
	Role            domainauth.Role
}

// CurrentSession is the safe session context made available to handlers.
type CurrentSession struct {
	ID        string
	DeviceID  string
	ExpiresAt time.Time
}

type authenticationContextKey struct{}

type authenticationContext struct {
	user    CurrentUser
	session CurrentSession
}

// AuthenticationMiddleware requires a valid bearer session and attaches its
// safe user/session context to the request.
func AuthenticationMiddleware(service BearerAuthenticationService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			token, ok := bearerToken(request)
			if !ok {
				writeUnauthenticated(response)
				return
			}

			result, err := service.Authenticate(request.Context(), token)
			switch {
			case errors.Is(err, applicationauth.ErrUnauthenticated):
				writeUnauthenticated(response)
				return
			case err != nil:
				WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
				return
			}

			value := authenticationContext{
				user: CurrentUser{
					ID:              result.User.ID,
					Name:            result.User.Name,
					EmailOrUsername: result.User.EmailOrUsername,
					Status:          result.User.Status,
					Role:            result.User.Role,
				},
				session: CurrentSession{
					ID:        result.Session.ID,
					DeviceID:  result.Session.DeviceID,
					ExpiresAt: result.Session.ExpiresAt,
				},
			}
			ctx := context.WithValue(request.Context(), authenticationContextKey{}, value)
			next.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

// HasPermission lets handlers and application adapters ask for a concrete
// capability without branching on role names.
func (user CurrentUser) HasPermission(permission domainauth.Permission) bool {
	return domainauth.RoleHasPermission(user.Role, permission)
}

// CurrentUserFromContext returns the authenticated, password-hash-free user.
func CurrentUserFromContext(ctx context.Context) (CurrentUser, bool) {
	value, ok := ctx.Value(authenticationContextKey{}).(authenticationContext)
	return value.user, ok
}

// CurrentSessionFromContext returns safe metadata for the bearer session.
func CurrentSessionFromContext(ctx context.Context) (CurrentSession, bool) {
	value, ok := ctx.Value(authenticationContextKey{}).(authenticationContext)
	return value.session, ok
}

func bearerToken(request *http.Request) (string, bool) {
	values := request.Header.Values("Authorization")
	if len(values) != 1 {
		return "", false
	}
	parts := strings.Fields(values[0])
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func writeUnauthenticated(response http.ResponseWriter) {
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("WWW-Authenticate", "Bearer")
	WriteError(response, http.StatusUnauthorized, "unauthenticated", "Authentication required", nil)
}
