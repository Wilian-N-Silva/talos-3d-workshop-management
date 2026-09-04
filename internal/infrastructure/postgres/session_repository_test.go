package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
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
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE maintenance_events, job_labor_entries, labor_rates, energy_measurements, print_job_material_usage, job_events, print_jobs, catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, workshop_settings, files, sessions, bootstrap_state, users, client_devices"); err != nil {
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

	plaintextToken := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xa5}, 32))
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

	found, err := repository.FindByID(ctx, created.ID)
	if err != nil || found.ID != created.ID {
		t.Fatalf("FindByID() = %#v, %v", found, err)
	}
	if _, err := repository.FindByID(ctx, "00000000-0000-4000-8000-000000000000"); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("FindByID() missing error = %v, want ErrSessionNotFound", err)
	}
	details, err := repository.ListByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListByUserID() error = %v", err)
	}
	if len(details) != 1 || details[0].Session.ID != created.ID || details[0].Device.ID != device.ID || details[0].Device.DisplayName != "Workshop PC" {
		t.Fatalf("ListByUserID() = %#v", details)
	}
	if details[0].Session.TokenHash != nil {
		t.Fatalf("ListByUserID() exposed token hash = %x", details[0].Session.TokenHash)
	}
	emptyDetails, err := repository.ListByUserID(ctx, "00000000-0000-4000-8000-000000000000")
	if err != nil || emptyDetails == nil || len(emptyDetails) != 0 {
		t.Fatalf("ListByUserID() empty = %#v, %v", emptyDetails, err)
	}

	authentication, err := applicationauth.NewAuthenticationService(repository, applicationauth.DefaultSessionLastUsedInterval)
	if err != nil {
		t.Fatalf("NewAuthenticationService() error = %v", err)
	}
	if _, err := authentication.Authenticate(ctx, plaintextToken); err != nil {
		t.Fatalf("Authenticate() before revocation error = %v", err)
	}
	revokedAt := time.Now().UTC().Truncate(time.Microsecond)
	revoked, err := repository.Revoke(ctx, created.ID, revokedAt)
	if err != nil || revoked.RevokedAt == nil || !revoked.RevokedAt.Equal(revokedAt) {
		t.Fatalf("Revoke() = %#v, %v", revoked, err)
	}
	secondRevocation := revokedAt.Add(time.Minute)
	revokedAgain, err := repository.Revoke(ctx, created.ID, secondRevocation)
	if err != nil || revokedAgain.RevokedAt == nil || !revokedAgain.RevokedAt.Equal(revokedAt) {
		t.Fatalf("Revoke() repeated = %#v, %v, want original %s", revokedAgain, err, revokedAt)
	}
	if _, err := authentication.Authenticate(ctx, plaintextToken); !errors.Is(err, applicationauth.ErrUnauthenticated) {
		t.Fatalf("Authenticate() after revocation error = %v, want ErrUnauthenticated", err)
	}
	if _, err := repository.Revoke(ctx, "00000000-0000-4000-8000-000000000000", revokedAt); !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("Revoke() missing error = %v, want ErrSessionNotFound", err)
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
