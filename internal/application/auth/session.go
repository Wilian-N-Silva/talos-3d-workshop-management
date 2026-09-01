package auth

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const sessionTokenBytes = 32

// ErrInvalidSessionExpiry indicates that a session would not be valid after issuance.
var ErrInvalidSessionExpiry = errors.New("invalid session expiry")

// SessionRepository persists hash-only session records.
type SessionRepository interface {
	Create(context.Context, domainauth.CreateSessionParams) (domainauth.Session, error)
}

// CreateSessionInput identifies the user, desktop installation, and explicit
// expiry for a new session.
type CreateSessionInput struct {
	UserID    string
	DeviceID  string
	ExpiresAt time.Time
}

// IssuedSession contains the persisted record and the bearer token that is
// returned only at issuance time. Token must never be persisted server-side.
type IssuedSession struct {
	Session domainauth.Session
	Token   string
}

// SessionService securely issues opaque desktop sessions.
type SessionService struct {
	repository SessionRepository
	random     io.Reader
	now        func() time.Time
}

// NewSessionService creates a session issuer backed by cryptographic randomness.
func NewSessionService(repository SessionRepository) *SessionService {
	return newSessionService(repository, cryptorand.Reader, time.Now)
}

func newSessionService(
	repository SessionRepository,
	random io.Reader,
	now func() time.Time,
) *SessionService {
	return &SessionService{repository: repository, random: random, now: now}
}

// Create generates a 256-bit opaque token, persists only its SHA-256 digest,
// and returns the plaintext token once with the created session.
func (service *SessionService) Create(
	ctx context.Context,
	input CreateSessionInput,
) (IssuedSession, error) {
	expiresAt := input.ExpiresAt.UTC()
	if !expiresAt.After(service.now().UTC()) {
		return IssuedSession{}, ErrInvalidSessionExpiry
	}

	tokenBytes := make([]byte, sessionTokenBytes)
	defer clear(tokenBytes)
	if _, err := io.ReadFull(service.random, tokenBytes); err != nil {
		return IssuedSession{}, fmt.Errorf("generate session token: %w", err)
	}

	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	tokenHash := HashSessionToken(token)
	session, err := service.repository.Create(ctx, domainauth.CreateSessionParams{
		UserID:    input.UserID,
		DeviceID:  input.DeviceID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return IssuedSession{}, fmt.Errorf("persist session: %w", err)
	}

	return IssuedSession{Session: session, Token: token}, nil
}

// HashSessionToken derives the lookup-safe SHA-256 digest used by persistence.
func HashSessionToken(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	hash := make([]byte, len(digest))
	copy(hash, digest[:])
	return hash
}
