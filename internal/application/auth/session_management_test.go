package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const (
	testActorUserID  = "11111111-1111-4111-8111-111111111111"
	testTargetUserID = "22222222-2222-4222-8222-222222222222"
	testSessionID    = "33333333-3333-4333-8333-333333333333"
)

func TestSessionManagementListsOwnSessions(t *testing.T) {
	want := []domainauth.SessionDetails{{Session: domainauth.Session{ID: testSessionID, UserID: testActorUserID}}}
	repository := &sessionManagementRepositoryStub{listResult: want}
	service := newTestSessionManagementService(t, repository, time.Now())

	got, err := service.List(context.Background(), SessionActor{UserID: testActorUserID, Role: domainauth.RoleViewer}, "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 || got[0].Session.ID != testSessionID || repository.listUserID != testActorUserID {
		t.Fatalf("List() = %#v, target = %q", got, repository.listUserID)
	}
}

func TestSessionManagementAdminPermissionTargetsAnotherUser(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		repository := &sessionManagementRepositoryStub{}
		service := newTestSessionManagementService(t, repository, time.Now())
		if _, err := service.List(context.Background(), SessionActor{UserID: testActorUserID, Role: domainauth.RoleOwner}, testTargetUserID); err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if repository.listUserID != testTargetUserID {
			t.Fatalf("list target = %q, want %q", repository.listUserID, testTargetUserID)
		}
	})

	t.Run("revoke", func(t *testing.T) {
		now := time.Date(2026, time.September, 2, 13, 30, 0, 0, time.UTC)
		repository := &sessionManagementRepositoryStub{found: domainauth.Session{ID: testSessionID, UserID: testTargetUserID}}
		service := newTestSessionManagementService(t, repository, now)
		if _, err := service.Revoke(context.Background(), SessionActor{UserID: testActorUserID, Role: domainauth.RoleOwner}, testSessionID); err != nil {
			t.Fatalf("Revoke() error = %v", err)
		}
		if repository.revokeID != testSessionID || !repository.revokedAt.Equal(now) {
			t.Fatalf("revoke = %q at %s", repository.revokeID, repository.revokedAt)
		}
	})
}

func TestSessionManagementDeniesAnotherUserWithoutPermission(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		repository := &sessionManagementRepositoryStub{}
		service := newTestSessionManagementService(t, repository, time.Now())
		_, err := service.List(context.Background(), SessionActor{UserID: testActorUserID, Role: domainauth.RoleViewer}, testTargetUserID)
		if !errors.Is(err, ErrSessionAccessDenied) || repository.listCalls != 0 {
			t.Fatalf("List() error = %v, calls = %d", err, repository.listCalls)
		}
	})

	t.Run("revoke", func(t *testing.T) {
		repository := &sessionManagementRepositoryStub{found: domainauth.Session{ID: testSessionID, UserID: testTargetUserID}}
		service := newTestSessionManagementService(t, repository, time.Now())
		_, err := service.Revoke(context.Background(), SessionActor{UserID: testActorUserID, Role: domainauth.RoleViewer}, testSessionID)
		if !errors.Is(err, ErrSessionAccessDenied) || repository.revokeCalls != 0 {
			t.Fatalf("Revoke() error = %v, revoke calls = %d", err, repository.revokeCalls)
		}
	})
}

func TestSessionManagementRevokesOwnSession(t *testing.T) {
	now := time.Date(2026, time.September, 2, 13, 45, 0, 0, time.UTC)
	repository := &sessionManagementRepositoryStub{found: domainauth.Session{ID: testSessionID, UserID: testActorUserID}}
	service := newTestSessionManagementService(t, repository, now)

	got, err := service.Revoke(context.Background(), SessionActor{UserID: testActorUserID, Role: domainauth.RoleViewer}, testSessionID)
	if err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if got.ID != testSessionID || repository.findID != testSessionID || repository.revokeCalls != 1 {
		t.Fatalf("Revoke() = %#v, repository = %#v", got, repository)
	}
}

func TestSessionManagementValidatesIDsBeforeRepositoryAccess(t *testing.T) {
	repository := &sessionManagementRepositoryStub{}
	service := newTestSessionManagementService(t, repository, time.Now())

	if _, err := service.List(context.Background(), SessionActor{UserID: "not-a-uuid", Role: domainauth.RoleOwner}, ""); !errors.Is(err, ErrInvalidSessionManagementInput) {
		t.Fatalf("List() error = %v", err)
	}
	if _, err := service.Revoke(context.Background(), SessionActor{UserID: testActorUserID, Role: domainauth.RoleOwner}, "not-a-uuid"); !errors.Is(err, ErrInvalidSessionManagementInput) {
		t.Fatalf("Revoke() error = %v", err)
	}
	if repository.listCalls != 0 || repository.findCalls != 0 {
		t.Fatalf("repository calls = list %d, find %d", repository.listCalls, repository.findCalls)
	}
}

func TestSessionManagementPreservesNotFoundAndWrapsDependencies(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		repository := &sessionManagementRepositoryStub{findError: domainauth.ErrSessionNotFound}
		service := newTestSessionManagementService(t, repository, time.Now())
		_, err := service.Revoke(context.Background(), SessionActor{UserID: testActorUserID}, testSessionID)
		if !errors.Is(err, domainauth.ErrSessionNotFound) {
			t.Fatalf("Revoke() error = %v", err)
		}
	})

	t.Run("list dependency", func(t *testing.T) {
		dependencyError := errors.New("database unavailable")
		repository := &sessionManagementRepositoryStub{listError: dependencyError}
		service := newTestSessionManagementService(t, repository, time.Now())
		_, err := service.List(context.Background(), SessionActor{UserID: testActorUserID}, "")
		if !errors.Is(err, dependencyError) {
			t.Fatalf("List() error = %v", err)
		}
	})
}

func TestNewSessionManagementServiceRejectsInvalidConfiguration(t *testing.T) {
	if _, err := NewSessionManagementService(nil); !errors.Is(err, ErrInvalidSessionManagementConfiguration) {
		t.Fatalf("nil repository error = %v", err)
	}
	if _, err := newSessionManagementService(&sessionManagementRepositoryStub{}, nil); !errors.Is(err, ErrInvalidSessionManagementConfiguration) {
		t.Fatalf("nil clock error = %v", err)
	}
}

func newTestSessionManagementService(t *testing.T, repository SessionManagementRepository, now time.Time) *SessionManagementService {
	t.Helper()
	service, err := newSessionManagementService(repository, func() time.Time { return now })
	if err != nil {
		t.Fatalf("newSessionManagementService() error = %v", err)
	}
	return service
}

type sessionManagementRepositoryStub struct {
	listResult  []domainauth.SessionDetails
	listError   error
	listUserID  string
	listCalls   int
	found       domainauth.Session
	findError   error
	findID      string
	findCalls   int
	revokeError error
	revokeID    string
	revokedAt   time.Time
	revokeCalls int
}

func (stub *sessionManagementRepositoryStub) ListByUserID(_ context.Context, userID string) ([]domainauth.SessionDetails, error) {
	stub.listCalls++
	stub.listUserID = userID
	return stub.listResult, stub.listError
}

func (stub *sessionManagementRepositoryStub) FindByID(_ context.Context, id string) (domainauth.Session, error) {
	stub.findCalls++
	stub.findID = id
	return stub.found, stub.findError
}

func (stub *sessionManagementRepositoryStub) Revoke(_ context.Context, id string, revokedAt time.Time) (domainauth.Session, error) {
	stub.revokeCalls++
	stub.revokeID = id
	stub.revokedAt = revokedAt
	result := stub.found
	result.RevokedAt = &revokedAt
	return result, stub.revokeError
}
