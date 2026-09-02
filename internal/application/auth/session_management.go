package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

var (
	// ErrInvalidSessionManagementInput indicates a malformed user or session ID.
	ErrInvalidSessionManagementInput = errors.New("invalid session management input")
	// ErrSessionAccessDenied indicates that an actor cannot manage the target user's sessions.
	ErrSessionAccessDenied = errors.New("session access denied")
	// ErrInvalidSessionManagementConfiguration indicates missing service dependencies.
	ErrInvalidSessionManagementConfiguration = errors.New("invalid session management configuration")
)

// SessionManagementRepository provides session audit listing and revocation.
type SessionManagementRepository interface {
	ListByUserID(context.Context, string) ([]domainauth.SessionDetails, error)
	FindByID(context.Context, string) (domainauth.Session, error)
	Revoke(context.Context, string, time.Time) (domainauth.Session, error)
}

// SessionActor is the authenticated identity requesting session management.
type SessionActor struct {
	UserID string
	Role   domainauth.Role
}

// SessionManagementService applies ownership and users.manage authorization.
type SessionManagementService struct {
	repository SessionManagementRepository
	now        func() time.Time
}

// NewSessionManagementService creates the application boundary for session listing and revocation.
func NewSessionManagementService(repository SessionManagementRepository) (*SessionManagementService, error) {
	return newSessionManagementService(repository, time.Now)
}

func newSessionManagementService(
	repository SessionManagementRepository,
	now func() time.Time,
) (*SessionManagementService, error) {
	if repository == nil || now == nil {
		return nil, ErrInvalidSessionManagementConfiguration
	}
	return &SessionManagementService{repository: repository, now: now}, nil
}

// List returns sessions for the actor, or for another user when the actor has users.manage.
func (service *SessionManagementService) List(
	ctx context.Context,
	actor SessionActor,
	requestedUserID string,
) ([]domainauth.SessionDetails, error) {
	targetUserID, err := service.authorizeTarget(actor, requestedUserID)
	if err != nil {
		return nil, err
	}
	sessions, err := service.repository.ListByUserID(ctx, targetUserID)
	if err != nil {
		return nil, fmt.Errorf("list user sessions: %w", err)
	}
	return sessions, nil
}

// Revoke marks a session inactive. Owners may revoke their own sessions and
// actors with users.manage may revoke any user's session.
func (service *SessionManagementService) Revoke(
	ctx context.Context,
	actor SessionActor,
	sessionID string,
) (domainauth.Session, error) {
	if !validUUID(actor.UserID) || !validUUID(sessionID) {
		return domainauth.Session{}, ErrInvalidSessionManagementInput
	}

	session, err := service.repository.FindByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, domainauth.ErrSessionNotFound) {
			return domainauth.Session{}, domainauth.ErrSessionNotFound
		}
		return domainauth.Session{}, fmt.Errorf("find session for revocation: %w", err)
	}
	if session.UserID != actor.UserID && !domainauth.RoleHasPermission(actor.Role, domainauth.PermissionUsersManage) {
		return domainauth.Session{}, ErrSessionAccessDenied
	}

	revoked, err := service.repository.Revoke(ctx, session.ID, service.now().UTC())
	if err != nil {
		if errors.Is(err, domainauth.ErrSessionNotFound) {
			return domainauth.Session{}, domainauth.ErrSessionNotFound
		}
		return domainauth.Session{}, fmt.Errorf("revoke session: %w", err)
	}
	return revoked, nil
}

func (service *SessionManagementService) authorizeTarget(actor SessionActor, requestedUserID string) (string, error) {
	if !validUUID(actor.UserID) {
		return "", ErrInvalidSessionManagementInput
	}
	targetUserID := strings.TrimSpace(requestedUserID)
	if targetUserID == "" {
		targetUserID = actor.UserID
	}
	if !validUUID(targetUserID) {
		return "", ErrInvalidSessionManagementInput
	}
	if targetUserID != actor.UserID && !domainauth.RoleHasPermission(actor.Role, domainauth.PermissionUsersManage) {
		return "", ErrSessionAccessDenied
	}
	return targetUserID, nil
}
