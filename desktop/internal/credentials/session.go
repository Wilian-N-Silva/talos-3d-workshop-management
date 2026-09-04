// Package credentials stores desktop secrets in Windows Credential Manager.
package credentials

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const credentialTargetPrefix = "TalosWorkshopManagement/session/"

// Session is the native-only persisted authentication state. It must never be
// returned directly through a Wails binding because it contains the bearer token.
type Session struct {
	Token           string    `json:"token"`
	ExpiresAt       time.Time `json:"expires_at"`
	UserID          string    `json:"user_id"`
	UserName        string    `json:"user_name"`
	EmailOrUsername string    `json:"email_or_username"`
}

type backend interface {
	Write(target string, secret []byte) error
	Read(target string) ([]byte, error)
	Delete(target string) error
}

// ErrNotFound indicates that no credential exists for the selected server.
var ErrNotFound = errors.New("credential not found")

// Store persists sessions under a server-specific target.
type Store struct {
	backend backend
	now     func() time.Time
}

// NewStore creates the production Windows Credential Manager store.
func NewStore() *Store {
	return &Store{backend: windowsCredentialBackend{}, now: time.Now}
}

func newStore(backend backend, now func() time.Time) *Store {
	return &Store{backend: backend, now: now}
}

// Save validates and securely persists a session.
func (store *Store) Save(serverBaseURL string, session Session) error {
	if strings.TrimSpace(session.Token) == "" || strings.TrimSpace(session.UserID) == "" ||
		strings.TrimSpace(session.UserName) == "" || strings.TrimSpace(session.EmailOrUsername) == "" ||
		session.ExpiresAt.IsZero() {
		return errors.New("invalid session credential")
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("encode session credential: %w", err)
	}
	defer clear(payload)
	if err := store.backend.Write(credentialTarget(serverBaseURL), payload); err != nil {
		return fmt.Errorf("write session credential: %w", err)
	}
	return nil
}

// Load returns a non-expired session. Expired credentials are removed.
func (store *Store) Load(serverBaseURL string) (Session, error) {
	payload, err := store.backend.Read(credentialTarget(serverBaseURL))
	if err != nil {
		return Session{}, err
	}
	defer clear(payload)
	var session Session
	if err := json.Unmarshal(payload, &session); err != nil {
		return Session{}, fmt.Errorf("decode session credential: %w", err)
	}
	if strings.TrimSpace(session.Token) == "" || session.ExpiresAt.IsZero() {
		return Session{}, errors.New("invalid session credential")
	}
	if !session.ExpiresAt.After(store.now().UTC()) {
		if err := store.Delete(serverBaseURL); err != nil {
			return Session{}, err
		}
		return Session{}, ErrNotFound
	}
	return session, nil
}

// Delete removes the session for a server and is idempotent.
func (store *Store) Delete(serverBaseURL string) error {
	if err := store.backend.Delete(credentialTarget(serverBaseURL)); err != nil && !errors.Is(err, ErrNotFound) {
		return fmt.Errorf("delete session credential: %w", err)
	}
	return nil
}

func credentialTarget(serverBaseURL string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(serverBaseURL)))
	return credentialTargetPrefix + hex.EncodeToString(digest[:])
}
