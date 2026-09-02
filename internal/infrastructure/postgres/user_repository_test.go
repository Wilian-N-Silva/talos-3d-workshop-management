package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

func TestUserRepositoryAgainstPostgreSQL(t *testing.T) {
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
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE sessions, bootstrap_state, users"); err != nil {
		t.Fatalf("truncate user bootstrap tables: %v", err)
	}

	repository := NewUserRepository(database)
	created, err := repository.Create(ctx, auth.CreateUserParams{
		Name:            "Workshop Owner",
		EmailOrUsername: "owner@example.com",
		PasswordHash:    "$argon2id$test-hash",
		Status:          auth.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidPattern.MatchString(created.ID) {
		t.Fatalf("created ID = %q, want UUID", created.ID)
	}
	if created.Name != "Workshop Owner" || created.EmailOrUsername != "owner@example.com" {
		t.Fatalf("created user identity fields = %#v", created)
	}
	if created.PasswordHash != "$argon2id$test-hash" || created.Status != auth.UserStatusActive {
		t.Fatalf("created authentication fields = %#v", created)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() || created.UpdatedAt.Before(created.CreatedAt) {
		t.Fatalf("created timestamps = %s, %s", created.CreatedAt, created.UpdatedAt)
	}
	if _, offset := created.CreatedAt.Zone(); offset != 0 {
		t.Fatalf("created timestamp offset = %d, want UTC", offset)
	}
	if created.LastLoginAt != nil {
		t.Fatalf("LastLoginAt = %s, want nil", created.LastLoginAt)
	}

	found, err := repository.FindByEmailOrUsername(ctx, "OWNER@EXAMPLE.COM")
	if err != nil {
		t.Fatalf("FindByEmailOrUsername() error = %v", err)
	}
	if found.ID != created.ID || found.EmailOrUsername != created.EmailOrUsername {
		t.Fatalf("found user = %#v, want ID %q", found, created.ID)
	}

	loggedInAt := created.CreatedAt.Add(time.Hour)
	updated, err := repository.UpdateLastLogin(ctx, created.ID, loggedInAt)
	if err != nil {
		t.Fatalf("UpdateLastLogin() error = %v", err)
	}
	if updated.LastLoginAt == nil || !updated.LastLoginAt.Equal(loggedInAt) {
		t.Fatalf("updated LastLoginAt = %v, want %s", updated.LastLoginAt, loggedInAt)
	}
	if _, err := repository.UpdateLastLogin(ctx, created.ID, created.CreatedAt.Add(time.Minute)); err != nil {
		t.Fatalf("UpdateLastLogin() delayed error = %v", err)
	}
	refound, err := repository.FindByEmailOrUsername(ctx, created.EmailOrUsername)
	if err != nil {
		t.Fatalf("FindByEmailOrUsername() after update error = %v", err)
	}
	if refound.LastLoginAt == nil || !refound.LastLoginAt.Equal(loggedInAt) {
		t.Fatalf("monotonic LastLoginAt = %v, want %s", refound.LastLoginAt, loggedInAt)
	}

	if _, err := repository.Create(ctx, auth.CreateUserParams{
		Name:            "Duplicate Owner",
		EmailOrUsername: "Owner@Example.com",
		PasswordHash:    "$argon2id$other-test-hash",
		Status:          auth.UserStatusActive,
	}); err == nil {
		t.Fatal("Create() duplicate login error = nil")
	}

	if _, err := repository.Create(ctx, auth.CreateUserParams{
		Name:            "Invalid Status",
		EmailOrUsername: "invalid-status",
		PasswordHash:    "$argon2id$test-hash",
		Status:          auth.UserStatus("pending"),
	}); err == nil {
		t.Fatal("Create() invalid status error = nil")
	}

	count, err := repository.Count(ctx)
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("Count() = %d, want 1", count)
	}
	if needsSetup, err := repository.NeedsSetup(ctx); err != nil || needsSetup {
		t.Fatalf("NeedsSetup() with existing user = %t, %v, want false", needsSetup, err)
	}

	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE sessions, bootstrap_state, users"); err != nil {
		t.Fatalf("reset user bootstrap tables: %v", err)
	}
	if needsSetup, err := repository.NeedsSetup(ctx); err != nil || !needsSetup {
		t.Fatalf("NeedsSetup() empty database = %t, %v, want true", needsSetup, err)
	}
	if _, err := repository.FindByEmailOrUsername(ctx, "missing"); !errors.Is(err, auth.ErrUserNotFound) {
		t.Fatalf("FindByEmailOrUsername() missing error = %v, want ErrUserNotFound", err)
	}

	type creationResult struct {
		user auth.User
		err  error
	}
	const attempts = 8
	start := make(chan struct{})
	results := make(chan creationResult, attempts)
	for index := range attempts {
		go func() {
			<-start
			user, err := repository.CreateFirst(ctx, auth.CreateUserParams{
				Name:            fmt.Sprintf("Owner %d", index),
				EmailOrUsername: fmt.Sprintf("owner-%d", index),
				PasswordHash:    "$argon2id$concurrent-test-hash",
				Status:          auth.UserStatusActive,
			})
			results <- creationResult{user: user, err: err}
		}()
	}
	close(start)

	successes := 0
	var initialOwnerID string
	for range attempts {
		result := <-results
		switch {
		case result.err == nil:
			successes++
			initialOwnerID = result.user.ID
		case errors.Is(result.err, auth.ErrFirstUserAlreadyExists):
		default:
			t.Fatalf("CreateFirst() concurrent error = %v", result.err)
		}
	}
	if successes != 1 {
		t.Fatalf("CreateFirst() successes = %d, want 1", successes)
	}
	if needsSetup, err := repository.NeedsSetup(ctx); err != nil || needsSetup {
		t.Fatalf("NeedsSetup() after creation = %t, %v, want false", needsSetup, err)
	}
	if count, err := repository.Count(ctx); err != nil || count != 1 {
		t.Fatalf("Count() after concurrent creation = %d, %v, want 1", count, err)
	}

	var recordedOwnerID string
	if err := database.QueryRowContext(ctx, "SELECT initial_owner_user_id FROM bootstrap_state").Scan(&recordedOwnerID); err != nil {
		t.Fatalf("read bootstrap owner marker: %v", err)
	}
	if recordedOwnerID != initialOwnerID {
		t.Fatalf("recorded initial owner ID = %q, want %q", recordedOwnerID, initialOwnerID)
	}
	if _, err := database.ExecContext(ctx, "DELETE FROM users WHERE id = $1", initialOwnerID); err == nil {
		t.Fatal("deleting the recorded initial owner succeeded and could reopen bootstrap")
	}
}
