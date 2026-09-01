package postgres

import (
	"context"
	"os"
	"regexp"
	"testing"

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
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE users"); err != nil {
		t.Fatalf("truncate users: %v", err)
	}

	repository := NewUserRepository(database)
	created, err := repository.Create(ctx, CreateUserParams{
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

	if _, err := repository.Create(ctx, CreateUserParams{
		Name:            "Duplicate Owner",
		EmailOrUsername: "Owner@Example.com",
		PasswordHash:    "$argon2id$other-test-hash",
		Status:          auth.UserStatusActive,
	}); err == nil {
		t.Fatal("Create() duplicate login error = nil")
	}

	if _, err := repository.Create(ctx, CreateUserParams{
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
}
