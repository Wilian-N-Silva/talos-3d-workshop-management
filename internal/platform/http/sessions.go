package httpplatform

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const (
	SessionsPath      = "/auth/sessions"
	sessionRevokePath = SessionsPath + "/{sessionID}/revoke"
)

// SessionManagementService lists and revokes sessions under application-layer authorization.
type SessionManagementService interface {
	List(context.Context, applicationauth.SessionActor, string) ([]domainauth.SessionDetails, error)
	Revoke(context.Context, applicationauth.SessionActor, string) (domainauth.Session, error)
}

type sessionListResponse struct {
	Sessions []sessionResponse `json:"sessions"`
}

type sessionResponse struct {
	ID         string                `json:"id"`
	UserID     string                `json:"user_id"`
	Device     sessionDeviceResponse `json:"device"`
	CreatedAt  time.Time             `json:"created_at"`
	ExpiresAt  time.Time             `json:"expires_at"`
	LastUsedAt *time.Time            `json:"last_used_at"`
	RevokedAt  *time.Time            `json:"revoked_at"`
	Current    bool                  `json:"current"`
}

type sessionDeviceResponse struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	OS          string    `json:"os"`
	AppVersion  string    `json:"app_version"`
	LastSeenAt  time.Time `json:"last_seen_at"`
}

// RegisterSessionManagement registers bearer-protected self-service and
// users.manage session endpoints.
func RegisterSessionManagement(
	router *APIV1Router,
	authentication BearerAuthenticationService,
	sessions SessionManagementService,
) {
	authenticate := AuthenticationMiddleware(authentication)

	router.Handle(http.MethodGet, SessionsPath, authenticate(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		user, session, ok := sessionRequestContext(request.Context())
		if !ok {
			writeUnauthenticated(response)
			return
		}

		details, err := sessions.List(request.Context(), sessionActor(user), request.URL.Query().Get("user_id"))
		if writeSessionManagementError(response, err) {
			return
		}

		items := make([]sessionResponse, 0, len(details))
		for _, detail := range details {
			items = append(items, sessionResponse{
				ID:         detail.Session.ID,
				UserID:     detail.Session.UserID,
				CreatedAt:  detail.Session.CreatedAt,
				ExpiresAt:  detail.Session.ExpiresAt,
				LastUsedAt: detail.Session.LastUsedAt,
				RevokedAt:  detail.Session.RevokedAt,
				Current:    detail.Session.ID == session.ID,
				Device: sessionDeviceResponse{
					ID:          detail.Device.ID,
					DisplayName: detail.Device.DisplayName,
					OS:          detail.Device.OS,
					AppVersion:  detail.Device.AppVersion,
					LastSeenAt:  detail.Device.LastSeenAt,
				},
			})
		}
		writeJSON(response, http.StatusOK, sessionListResponse{Sessions: items})
	})))

	router.Handle(http.MethodPost, sessionRevokePath, authenticate(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		user, _, ok := sessionRequestContext(request.Context())
		if !ok {
			writeUnauthenticated(response)
			return
		}

		_, err := sessions.Revoke(request.Context(), sessionActor(user), chi.URLParam(request, "sessionID"))
		if writeSessionManagementError(response, err) {
			return
		}
		response.WriteHeader(http.StatusNoContent)
	})))
}

func sessionRequestContext(ctx context.Context) (CurrentUser, CurrentSession, bool) {
	user, userOK := CurrentUserFromContext(ctx)
	session, sessionOK := CurrentSessionFromContext(ctx)
	return user, session, userOK && sessionOK
}

func sessionActor(user CurrentUser) applicationauth.SessionActor {
	return applicationauth.SessionActor{UserID: user.ID, Role: user.Role}
}

func writeSessionManagementError(response http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, applicationauth.ErrInvalidSessionManagementInput):
		WriteError(response, http.StatusBadRequest, "invalid_request", "Invalid request", nil)
	case errors.Is(err, applicationauth.ErrSessionAccessDenied):
		WriteError(response, http.StatusForbidden, "forbidden", "Permission denied", nil)
	case errors.Is(err, domainauth.ErrSessionNotFound):
		WriteError(response, http.StatusNotFound, "session_not_found", "Session not found", nil)
	default:
		WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
	}
	return true
}
