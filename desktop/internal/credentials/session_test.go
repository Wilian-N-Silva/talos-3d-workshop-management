package credentials

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

func TestStoreRoundTripsAndDeletesSession(t *testing.T) {
	backend := &memoryBackend{secrets: map[string][]byte{}}
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	store := newStore(backend, func() time.Time { return now })
	session := Session{Token: "opaque-token", ExpiresAt: now.Add(time.Hour), UserID: "user-1", UserName: "Owner", EmailOrUsername: "owner"}

	if err := store.Save("http://workshop.local", session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	for _, secret := range backend.secrets {
		if !bytes.Contains(secret, []byte("opaque-token")) {
			t.Fatal("credential backend did not receive session payload")
		}
	}
	got, err := store.Load("http://workshop.local")
	if err != nil || got != session {
		t.Fatalf("Load() = %#v, %v", got, err)
	}
	if err := store.Delete("http://workshop.local"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := store.Load("http://workshop.local"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load() after delete error = %v", err)
	}
}

func TestStoreRemovesExpiredSession(t *testing.T) {
	backend := &memoryBackend{secrets: map[string][]byte{}}
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	store := newStore(backend, func() time.Time { return now })
	if err := store.Save("https://workshop.local", Session{Token: "expired", ExpiresAt: now.Add(-time.Second), UserID: "user-1", UserName: "Owner", EmailOrUsername: "owner"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := store.Load("https://workshop.local"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load() expired error = %v", err)
	}
	if len(backend.secrets) != 0 {
		t.Fatal("expired credential was not deleted")
	}
}

func TestCredentialTargetsAreServerSpecificAndDoNotExposeURL(t *testing.T) {
	first := credentialTarget("http://one.local")
	second := credentialTarget("http://two.local")
	if first == second || bytes.Contains([]byte(first), []byte("one.local")) {
		t.Fatalf("credential targets = %q and %q", first, second)
	}
}

type memoryBackend struct {
	secrets map[string][]byte
}

func (backend *memoryBackend) Write(target string, secret []byte) error {
	backend.secrets[target] = append([]byte(nil), secret...)
	return nil
}

func (backend *memoryBackend) Read(target string) ([]byte, error) {
	secret, ok := backend.secrets[target]
	if !ok {
		return nil, ErrNotFound
	}
	return append([]byte(nil), secret...), nil
}

func (backend *memoryBackend) Delete(target string) error {
	if _, ok := backend.secrets[target]; !ok {
		return ErrNotFound
	}
	delete(backend.secrets, target)
	return nil
}
