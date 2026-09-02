package postgres

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

func TestClientDeviceRepositoryAgainstPostgreSQL(t *testing.T) {
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
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE sessions, client_devices"); err != nil {
		t.Fatalf("truncate client devices: %v", err)
	}

	repository := NewClientDeviceRepository(database)
	created, err := repository.Create(ctx, auth.CreateClientDeviceParams{
		DisplayName: "Workshop PC",
		OS:          "Windows 11",
		AppVersion:  "1.0.0",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidPattern.MatchString(created.ID) {
		t.Fatalf("created ID = %q, want UUID", created.ID)
	}
	if created.DisplayName != "Workshop PC" || created.OS != "Windows 11" || created.AppVersion != "1.0.0" {
		t.Fatalf("created device metadata = %#v", created)
	}
	if created.CreatedAt.IsZero() || !created.LastSeenAt.Equal(created.CreatedAt) {
		t.Fatalf("created timestamps = %s, %s, want equal non-zero values", created.CreatedAt, created.LastSeenAt)
	}
	if _, offset := created.CreatedAt.Zone(); offset != 0 {
		t.Fatalf("created timestamp offset = %d, want UTC", offset)
	}

	found, err := repository.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found != created {
		t.Fatalf("FindByID() = %#v, want %#v", found, created)
	}

	observedAt := created.CreatedAt.Add(time.Hour)
	updated, err := repository.UpdateLastSeen(ctx, created.ID, observedAt)
	if err != nil {
		t.Fatalf("UpdateLastSeen() error = %v", err)
	}
	if !updated.LastSeenAt.Equal(observedAt) {
		t.Fatalf("updated LastSeenAt = %s, want %s", updated.LastSeenAt, observedAt)
	}

	delayed, err := repository.UpdateLastSeen(ctx, created.ID, created.CreatedAt.Add(time.Minute))
	if err != nil {
		t.Fatalf("UpdateLastSeen() delayed error = %v", err)
	}
	if !delayed.LastSeenAt.Equal(observedAt) {
		t.Fatalf("delayed LastSeenAt = %s, want unchanged %s", delayed.LastSeenAt, observedAt)
	}

	if _, err := repository.FindByID(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("FindByID() missing error = %v, want sql.ErrNoRows", err)
	}

	invalidValues := []auth.CreateClientDeviceParams{
		{DisplayName: " ", OS: "Windows 11", AppVersion: "1.0.0"},
		{DisplayName: "Workshop PC", OS: " Windows 11", AppVersion: "1.0.0"},
		{DisplayName: "Workshop PC", OS: "Windows 11", AppVersion: ""},
	}
	for _, params := range invalidValues {
		if _, err := repository.Create(ctx, params); err == nil {
			t.Fatalf("Create(%#v) error = nil, want validation error", params)
		}
	}
}
