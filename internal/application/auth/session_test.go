package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"testing"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

func TestSessionServiceCreatesOpaqueHashOnlySession(t *testing.T) {
	now := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.FixedZone("test", -3*60*60))
	expiresAt := now.Add(24 * time.Hour)
	randomBytes := make([]byte, sessionTokenBytes)
	for index := range randomBytes {
		randomBytes[index] = byte(index)
	}
	repository := &sessionRepositoryStub{}
	service := newSessionService(repository, bytes.NewReader(randomBytes), func() time.Time { return now })

	issued, err := service.Create(context.Background(), CreateSessionInput{
		UserID:    "user-id",
		DeviceID:  "device-id",
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	wantToken := base64.RawURLEncoding.EncodeToString(randomBytes)
	if issued.Token != wantToken || len(issued.Token) != 43 {
		t.Fatalf("issued token = %q, want 43-character base64url token %q", issued.Token, wantToken)
	}
	wantHash := sha256.Sum256([]byte(wantToken))
	if !bytes.Equal(repository.params.TokenHash, wantHash[:]) {
		t.Fatalf("persisted token hash = %x, want %x", repository.params.TokenHash, wantHash)
	}
	if bytes.Contains(repository.params.TokenHash, []byte(issued.Token)) {
		t.Fatal("persisted token hash contains plaintext token")
	}
	if repository.params.UserID != "user-id" || repository.params.DeviceID != "device-id" {
		t.Fatalf("persisted relationships = %#v", repository.params)
	}
	if !repository.params.ExpiresAt.Equal(expiresAt) || repository.params.ExpiresAt.Location() != time.UTC {
		t.Fatalf("persisted expiry = %s, want %s in UTC", repository.params.ExpiresAt, expiresAt.UTC())
	}
	if issued.Session.UserID != "user-id" || issued.Session.DeviceID != "device-id" {
		t.Fatalf("issued session = %#v", issued.Session)
	}
}

func TestSessionServiceGeneratesUniqueTokens(t *testing.T) {
	now := time.Date(2026, time.September, 1, 15, 0, 0, 0, time.UTC)
	randomBytes := append(bytes.Repeat([]byte{1}, sessionTokenBytes), bytes.Repeat([]byte{2}, sessionTokenBytes)...)
	service := newSessionService(&sessionRepositoryStub{}, bytes.NewReader(randomBytes), func() time.Time { return now })
	input := CreateSessionInput{UserID: "user-id", DeviceID: "device-id", ExpiresAt: now.Add(time.Hour)}

	first, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	second, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if first.Token == second.Token {
		t.Fatal("separate random inputs produced equal session tokens")
	}
}

func TestSessionServiceRejectsNonFutureExpiryBeforeGeneratingToken(t *testing.T) {
	now := time.Date(2026, time.September, 1, 15, 0, 0, 0, time.UTC)
	repository := &sessionRepositoryStub{}
	random := &countingReader{}
	service := newSessionService(repository, random, func() time.Time { return now })

	for _, expiresAt := range []time.Time{now, now.Add(-time.Nanosecond)} {
		issued, err := service.Create(context.Background(), CreateSessionInput{ExpiresAt: expiresAt})
		if !errors.Is(err, ErrInvalidSessionExpiry) {
			t.Fatalf("Create(%s) error = %v, want ErrInvalidSessionExpiry", expiresAt, err)
		}
		if issued.Token != "" || issued.Session.ID != "" {
			t.Fatalf("Create(%s) issued = %#v, want zero value", expiresAt, issued)
		}
	}
	if random.calls != 0 || repository.calls != 0 {
		t.Fatalf("invalid expiry reached dependencies: random calls %d, repository calls %d", random.calls, repository.calls)
	}
}

func TestSessionServiceDoesNotPersistWhenRandomGenerationFails(t *testing.T) {
	now := time.Now().UTC()
	repository := &sessionRepositoryStub{}
	service := newSessionService(repository, failingReader{}, func() time.Time { return now })

	issued, err := service.Create(context.Background(), CreateSessionInput{ExpiresAt: now.Add(time.Hour)})
	if err == nil || issued.Token != "" || issued.Session.ID != "" {
		t.Fatalf("Create() = %#v, %v, want zero value and error", issued, err)
	}
	if repository.calls != 0 {
		t.Fatalf("repository calls = %d, want 0", repository.calls)
	}
}

func TestSessionServiceDoesNotReturnTokenWhenPersistenceFails(t *testing.T) {
	now := time.Now().UTC()
	repository := &sessionRepositoryStub{err: errors.New("database unavailable")}
	service := newSessionService(repository, bytes.NewReader(make([]byte, sessionTokenBytes)), func() time.Time { return now })

	issued, err := service.Create(context.Background(), CreateSessionInput{ExpiresAt: now.Add(time.Hour)})
	if err == nil || issued.Token != "" || issued.Session.ID != "" {
		t.Fatalf("Create() = %#v, %v, want zero value and error", issued, err)
	}
}

type sessionRepositoryStub struct {
	params domainauth.CreateSessionParams
	err    error
	calls  int
}

func (stub *sessionRepositoryStub) Create(
	_ context.Context,
	params domainauth.CreateSessionParams,
) (domainauth.Session, error) {
	stub.calls++
	stub.params = params
	if stub.err != nil {
		return domainauth.Session{}, stub.err
	}
	return domainauth.Session{
		ID:        "session-id",
		UserID:    params.UserID,
		DeviceID:  params.DeviceID,
		TokenHash: append([]byte(nil), params.TokenHash...),
		ExpiresAt: params.ExpiresAt,
	}, nil
}

type countingReader struct {
	calls int
}

func (reader *countingReader) Read(buffer []byte) (int, error) {
	reader.calls++
	return len(buffer), nil
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
