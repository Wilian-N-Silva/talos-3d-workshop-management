package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

func TestAuthenticationServiceResolvesActiveSessionAndUpdatesLastUsed(t *testing.T) {
	now := time.Date(2026, time.September, 1, 15, 0, 0, 0, time.FixedZone("test", -3*60*60))
	token := validAuthenticationToken()
	repository := &authenticationRepositoryStub{
		session: domainauth.Session{ID: "session-id", DeviceID: "device-id", ExpiresAt: now.Add(time.Hour)},
		user:    domainauth.User{ID: "user-id", Status: domainauth.UserStatusActive},
		updated: true,
	}
	service := newTestAuthenticationService(t, repository, now)

	result, err := service.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if repository.findCalls != 1 || !bytes.Equal(repository.tokenHash, HashSessionToken(token)) {
		t.Fatalf("session lookup = %d calls with hash %x", repository.findCalls, repository.tokenHash)
	}
	if repository.updateCalls != 1 || repository.updateID != "session-id" {
		t.Fatalf("last-used update = %d calls for %q", repository.updateCalls, repository.updateID)
	}
	if !repository.observedAt.Equal(now.UTC()) || !repository.updateBefore.Equal(now.UTC().Add(-DefaultSessionLastUsedInterval)) {
		t.Fatalf("last-used update times = %s, %s", repository.observedAt, repository.updateBefore)
	}
	if result.Session.LastUsedAt == nil || !result.Session.LastUsedAt.Equal(now) || result.User.ID != "user-id" {
		t.Fatalf("authentication result = %#v", result)
	}
}

func TestAuthenticationServiceSkipsRecentLastUsedWrite(t *testing.T) {
	now := time.Date(2026, time.September, 1, 15, 0, 0, 0, time.UTC)
	recent := now.Add(-DefaultSessionLastUsedInterval + time.Second)
	repository := &authenticationRepositoryStub{
		session: domainauth.Session{ID: "session-id", ExpiresAt: now.Add(time.Hour), LastUsedAt: &recent},
		user:    domainauth.User{Status: domainauth.UserStatusActive},
	}
	service := newTestAuthenticationService(t, repository, now)

	if _, err := service.Authenticate(context.Background(), validAuthenticationToken()); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if repository.updateCalls != 0 {
		t.Fatalf("last-used update calls = %d, want 0", repository.updateCalls)
	}
}

func TestAuthenticationServiceRejectsInvalidSessionsUniformly(t *testing.T) {
	now := time.Date(2026, time.September, 1, 15, 0, 0, 0, time.UTC)
	revokedAt := now.Add(-time.Minute)
	tests := []struct {
		name      string
		session   domainauth.Session
		user      domainauth.User
		findError error
	}{
		{name: "missing", findError: domainauth.ErrSessionNotFound},
		{name: "expired", session: domainauth.Session{ExpiresAt: now}, user: domainauth.User{Status: domainauth.UserStatusActive}},
		{name: "revoked", session: domainauth.Session{ExpiresAt: now.Add(time.Hour), RevokedAt: &revokedAt}, user: domainauth.User{Status: domainauth.UserStatusActive}},
		{name: "disabled user", session: domainauth.Session{ExpiresAt: now.Add(time.Hour)}, user: domainauth.User{Status: domainauth.UserStatusDisabled}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &authenticationRepositoryStub{session: test.session, user: test.user, findError: test.findError}
			service := newTestAuthenticationService(t, repository, now)

			_, err := service.Authenticate(context.Background(), validAuthenticationToken())
			if !errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("Authenticate() error = %v, want ErrUnauthenticated", err)
			}
			if repository.updateCalls != 0 {
				t.Fatalf("last-used update calls = %d, want 0", repository.updateCalls)
			}
		})
	}
}

func TestAuthenticationServiceRejectsMalformedTokenBeforeLookup(t *testing.T) {
	tests := []string{
		"",
		"short",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa!",
		base64.RawStdEncoding.EncodeToString(bytes.Repeat([]byte{0xff}, sessionTokenBytes)),
		validAuthenticationToken() + " ",
	}
	for _, token := range tests {
		repository := &authenticationRepositoryStub{}
		service := newTestAuthenticationService(t, repository, time.Now())
		if _, err := service.Authenticate(context.Background(), token); !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("Authenticate(%q) error = %v, want ErrUnauthenticated", token, err)
		}
		if repository.findCalls != 0 {
			t.Fatalf("Authenticate(%q) lookup calls = %d, want 0", token, repository.findCalls)
		}
	}
}

func TestAuthenticationServiceReturnsDependencyErrors(t *testing.T) {
	now := time.Now().UTC()
	tests := []struct {
		name       string
		repository *authenticationRepositoryStub
	}{
		{name: "lookup", repository: &authenticationRepositoryStub{findError: errors.New("database unavailable")}},
		{name: "last used", repository: &authenticationRepositoryStub{
			session:     domainauth.Session{ID: "session-id", ExpiresAt: now.Add(time.Hour)},
			user:        domainauth.User{Status: domainauth.UserStatusActive},
			updateError: errors.New("database unavailable"),
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := newTestAuthenticationService(t, test.repository, now)
			if _, err := service.Authenticate(context.Background(), validAuthenticationToken()); err == nil || errors.Is(err, ErrUnauthenticated) {
				t.Fatalf("Authenticate() error = %v, want internal dependency error", err)
			}
		})
	}
}

func TestNewAuthenticationServiceRejectsInvalidConfiguration(t *testing.T) {
	if _, err := NewAuthenticationService(nil, time.Minute); !errors.Is(err, ErrInvalidAuthenticationConfiguration) {
		t.Fatalf("nil repository error = %v", err)
	}
	if _, err := NewAuthenticationService(&authenticationRepositoryStub{}, 0); !errors.Is(err, ErrInvalidAuthenticationConfiguration) {
		t.Fatalf("zero interval error = %v", err)
	}
}

func validAuthenticationToken() string {
	return base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xa5}, sessionTokenBytes))
}

func newTestAuthenticationService(
	t *testing.T,
	repository AuthenticationSessionRepository,
	now time.Time,
) *AuthenticationService {
	t.Helper()
	service, err := newAuthenticationService(repository, DefaultSessionLastUsedInterval, func() time.Time { return now })
	if err != nil {
		t.Fatalf("newAuthenticationService() error = %v", err)
	}
	return service
}

type authenticationRepositoryStub struct {
	session      domainauth.Session
	user         domainauth.User
	findError    error
	tokenHash    []byte
	findCalls    int
	updated      bool
	updateError  error
	updateID     string
	observedAt   time.Time
	updateBefore time.Time
	updateCalls  int
}

func (stub *authenticationRepositoryStub) FindByTokenHash(
	_ context.Context,
	tokenHash []byte,
) (domainauth.Session, domainauth.User, error) {
	stub.findCalls++
	stub.tokenHash = append([]byte(nil), tokenHash...)
	return stub.session, stub.user, stub.findError
}

func (stub *authenticationRepositoryStub) UpdateLastUsed(
	_ context.Context,
	id string,
	observedAt time.Time,
	updateBefore time.Time,
) (bool, error) {
	stub.updateCalls++
	stub.updateID = id
	stub.observedAt = observedAt
	stub.updateBefore = updateBefore
	return stub.updated, stub.updateError
}
