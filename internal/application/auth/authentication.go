package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const DefaultSessionLastUsedInterval = 5 * time.Minute

var (
	// ErrUnauthenticated is the uniform result for invalid or inactive sessions.
	ErrUnauthenticated = errors.New("unauthenticated")
	// ErrInvalidAuthenticationConfiguration indicates an unusable touch policy.
	ErrInvalidAuthenticationConfiguration = errors.New("invalid authentication configuration")
)

// AuthenticationSessionRepository resolves hash-only sessions and conditionally
// advances their last-used timestamp.
type AuthenticationSessionRepository interface {
	FindByTokenHash(context.Context, []byte) (domainauth.Session, domainauth.User, error)
	UpdateLastUsed(context.Context, string, time.Time, time.Time) (bool, error)
}

// AuthenticationResult is the active session and user resolved for a request.
type AuthenticationResult struct {
	Session domainauth.Session
	User    domainauth.User
}

// AuthenticationService validates bearer sessions and throttles audit writes.
type AuthenticationService struct {
	repository       AuthenticationSessionRepository
	lastUsedInterval time.Duration
	now              func() time.Time
}

// NewAuthenticationService creates a bearer-session resolver.
func NewAuthenticationService(
	repository AuthenticationSessionRepository,
	lastUsedInterval time.Duration,
) (*AuthenticationService, error) {
	return newAuthenticationService(repository, lastUsedInterval, time.Now)
}

func newAuthenticationService(
	repository AuthenticationSessionRepository,
	lastUsedInterval time.Duration,
	now func() time.Time,
) (*AuthenticationService, error) {
	if repository == nil || lastUsedInterval <= 0 || now == nil {
		return nil, ErrInvalidAuthenticationConfiguration
	}
	return &AuthenticationService{
		repository:       repository,
		lastUsedInterval: lastUsedInterval,
		now:              now,
	}, nil
}

// Authenticate resolves an opaque bearer token without persisting or returning
// it. Expired, revoked, missing, malformed, and disabled-user sessions share
// ErrUnauthenticated.
func (service *AuthenticationService) Authenticate(
	ctx context.Context,
	token string,
) (AuthenticationResult, error) {
	if !validSessionToken(token) {
		return AuthenticationResult{}, ErrUnauthenticated
	}

	session, user, err := service.repository.FindByTokenHash(ctx, HashSessionToken(token))
	if errors.Is(err, domainauth.ErrSessionNotFound) {
		return AuthenticationResult{}, ErrUnauthenticated
	}
	if err != nil {
		return AuthenticationResult{}, fmt.Errorf("resolve bearer session: %w", err)
	}

	now := service.now().UTC()
	if session.RevokedAt != nil || !now.Before(session.ExpiresAt) || user.Status != domainauth.UserStatusActive {
		return AuthenticationResult{}, ErrUnauthenticated
	}

	if shouldUpdateLastUsed(session.LastUsedAt, now, service.lastUsedInterval) {
		updated, err := service.repository.UpdateLastUsed(
			ctx,
			session.ID,
			now,
			now.Add(-service.lastUsedInterval),
		)
		if err != nil {
			return AuthenticationResult{}, fmt.Errorf("touch bearer session: %w", err)
		}
		if updated {
			session.LastUsedAt = &now
		}
	}

	return AuthenticationResult{Session: session, User: user}, nil
}

func validSessionToken(token string) bool {
	if strings.TrimSpace(token) != token || len(token) != 43 {
		return false
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil {
		return false
	}
	defer clear(decoded)
	return len(decoded) == sessionTokenBytes
}

func shouldUpdateLastUsed(lastUsedAt *time.Time, now time.Time, interval time.Duration) bool {
	return lastUsedAt == nil || now.Sub(lastUsedAt.UTC()) >= interval
}
