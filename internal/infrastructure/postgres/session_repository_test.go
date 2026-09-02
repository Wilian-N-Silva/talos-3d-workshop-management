package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

func TestSessionRepositoryAgainstPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("TALOS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TALOS_TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	database, err := Open(ctx, testConfig(databaseURL))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE sessions, bootstrap_state, users, client_devices"); err != nil {
		t.Fatalf("truncate authentication tables: %v", err)
	}

	user, err := NewUserRepository(database).Create(ctx, auth.CreateUserParams{
		Name:            "Workshop Owner",
		EmailOrUsername: "owner@example.com",
		PasswordHash:    "$argon2id$test-hash",
		Status:          auth.UserStatusActive,
		Role:            auth.RoleOwner,
	})
	if err != nil {
		t.Fatalf("create session user: %v", err)
	}
	device, err := NewClientDeviceRepository(database).Create(ctx, auth.CreateClientDeviceParams{
		DisplayName: "Workshop PC",
		OS:          "Windows 11",
		AppVersion:  "1.0.0",
	})
	if err != nil {
		t.Fatalf("create session device: %v", err)
	}

	plaintextToken := "server-must-not-persist-this-token"
	tokenHash := sha256.Sum256([]byte(plaintextToken))
	expiresAt := time.Now().UTC().Add(24 * time.Hour).Truncate(time.Microsecond)
	repository := NewSessionRepository(database)
	created, err := repository.Create(ctx, auth.CreateSessionParams{
		UserID:    user.ID,
		DeviceID:  device.ID,
		TokenHash: tokenHash[:],
		ExpiresAt: expiresAt,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidPattern.MatchString(created.ID) {
		t.Fatalf("created ID = %q, want UUID", created.ID)
	}
	if created.UserID != user.ID || created.DeviceID != device.ID {
		t.Fatalf("created relationships = %#v", created)
	}
	if !bytes.Equal(created.TokenHash, tokenHash[:]) {
		t.Fatalf("created token hash = %x, want %x", created.TokenHash, tokenHash)
	}
	if created.CreatedAt.IsZero() || !created.ExpiresAt.Equal(expiresAt) || !created.ExpiresAt.After(created.CreatedAt) {
		t.Fatalf("created timestamps = %s, %s", created.CreatedAt, created.ExpiresAt)
	}
	if created.LastUsedAt != nil || created.RevokedAt != nil {
		t.Fatalf("new session activity timestamps = %s, %s, want nil", created.LastUsedAt, created.RevokedAt)
	}
	if _, offset := created.CreatedAt.Zone(); offset != 0 {
		t.Fatalf("created timestamp offset = %d, want UTC", offset)
	}

	var storedHash []byte
	if err := database.QueryRowContext(ctx, "SELECT token_hash FROM sessions WHERE id = $1", created.ID).Scan(&storedHash); err != nil {
		t.Fatalf("read stored token hash: %v", err)
	}
	if !bytes.Equal(storedHash, tokenHash[:]) || bytes.Equal(storedHash, []byte(plaintextToken)) {
		t.Fatalf("stored token value = %x, want only SHA-256 hash", storedHash)
	}

	resolvedSession, resolvedUser, err := repository.FindByTokenHash(ctx, tokenHash[:])
	if err != nil {
		t.Fatalf("FindByTokenHash() error = %v", err)
	}
	if resolvedSession.ID != created.ID || resolvedUser.ID != user.ID || resolvedUser.PasswordHash != "$argon2id$test-hash" || resolvedUser.Role != auth.RoleOwner {
		t.Fatalf("resolved session/user = %#v, %#v", resolvedSession, resolvedUser)
	}
	missingHash := sha256.Sum256([]byte("missing-token"))
	if _, _, err := repository.FindByTokenHash(ctx, missingHash[:]); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("FindByTokenHash() missing error = %v, want ErrSessionNotFound", err)
	}

	firstUsedAt := created.CreatedAt.Add(time.Minute)
	updated, err := repository.UpdateLastUsed(ctx, created.ID, firstUsedAt, firstUsedAt.Add(-5*time.Minute))
	if err != nil || !updated {
		t.Fatalf("UpdateLastUsed() first = %t, %v, want true", updated, err)
	}
	secondUsedAt := firstUsedAt.Add(time.Minute)
	updated, err = repository.UpdateLastUsed(ctx, created.ID, secondUsedAt, secondUsedAt.Add(-5*time.Minute))
	if err != nil || updated {
		t.Fatalf("UpdateLastUsed() throttled = %t, %v, want false", updated, err)
	}

	concurrentUsedAt := created.CreatedAt.Add(10 * time.Minute)
	const attempts = 8
	results := make(chan bool, attempts)
	errorsFound := make(chan error, attempts)
	var wait sync.WaitGroup
	for range attempts {
		wait.Add(1)
		go func() {
			defer wait.Done()
			updated, err := repository.UpdateLastUsed(
				ctx,
				created.ID,
				concurrentUsedAt,
				concurrentUsedAt.Add(-5*time.Minute),
			)
			results <- updated
			errorsFound <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			t.Fatalf("UpdateLastUsed() concurrent error = %v", err)
		}
	}
	updates := 0
	for didUpdate := range results {
		if didUpdate {
			updates++
		}
	}
	if updates != 1 {
		t.Fatalf("concurrent last-used updates = %d, want 1", updates)
	}
	resolvedSession, _, err = repository.FindByTokenHash(ctx, tokenHash[:])
	if err != nil {
		t.Fatalf("FindByTokenHash() after update error = %v", err)
	}
	if resolvedSession.LastUsedAt == nil || !resolvedSession.LastUsedAt.Equal(concurrentUsedAt) {
		t.Fatalf("resolved LastUsedAt = %v, want %s", resolvedSession.LastUsedAt, concurrentUsedAt)
	}

	invalidDeviceHash := sha256.Sum256([]byte("invalid-device-token"))
	invalidExpiryHash := sha256.Sum256([]byte("invalid-expiry-token"))
	invalidSessions := []auth.CreateSessionParams{
		{UserID: user.ID, DeviceID: device.ID, TokenHash: []byte("short"), ExpiresAt: expiresAt},
		{UserID: user.ID, DeviceID: "00000000-0000-0000-0000-000000000000", TokenHash: invalidDeviceHash[:], ExpiresAt: expiresAt},
		{UserID: user.ID, DeviceID: device.ID, TokenHash: invalidExpiryHash[:], ExpiresAt: created.CreatedAt},
	}
	for _, params := range invalidSessions {
		if _, err := repository.Create(ctx, params); err == nil {
			t.Fatalf("Create(%#v) error = nil, want constraint error", params)
		}
	}
}
