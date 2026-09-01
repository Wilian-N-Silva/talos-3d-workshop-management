package postgres

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	migrationfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/migrations"
)

func TestMigrationStateIsCurrent(t *testing.T) {
	tests := []struct {
		name  string
		state MigrationState
		want  bool
	}{
		{name: "current", state: MigrationState{CurrentVersion: 1, TargetVersion: 1}, want: true},
		{name: "pending", state: MigrationState{CurrentVersion: 1, TargetVersion: 2, HasPending: true}},
		{name: "version mismatch without pending flag", state: MigrationState{CurrentVersion: 1, TargetVersion: 2}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.state.IsCurrent(); got != test.want {
				t.Fatalf("IsCurrent() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestMigrationLifecycleAgainstPostgreSQL(t *testing.T) {
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

	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS bootstrap_state"); err != nil {
		t.Fatalf("reset bootstrap state schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS client_devices"); err != nil {
		t.Fatalf("reset client devices schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS users"); err != nil {
		t.Fatalf("reset users schema: %v", err)
	}
	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS goose_db_version"); err != nil {
		t.Fatalf("reset migration state: %v", err)
	}

	state, err := GetMigrationState(ctx, database)
	if err != nil {
		t.Fatalf("GetMigrationState() before migrate error = %v", err)
	}
	if state.CurrentVersion != 0 || state.TargetVersion != 4 || !state.HasPending {
		t.Fatalf("state before migrate = %+v, want current 0, target 4, pending", state)
	}

	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	assertMigrationState(t, ctx, database, 4, 4, false)

	results := make(chan error, 2)
	for range 2 {
		go func() {
			results <- Migrate(ctx, database)
		}()
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("concurrent Migrate() error = %v", err)
		}
	}

	bootstrap, err := fs.ReadFile(migrationfiles.Files, "00001_bootstrap.sql")
	if err != nil {
		t.Fatalf("read bootstrap migration: %v", err)
	}
	users, err := fs.ReadFile(migrationfiles.Files, "00002_users.sql")
	if err != nil {
		t.Fatalf("read users migration: %v", err)
	}
	bootstrapState, err := fs.ReadFile(migrationfiles.Files, "00003_bootstrap_state.sql")
	if err != nil {
		t.Fatalf("read bootstrap state migration: %v", err)
	}
	clientDevices, err := fs.ReadFile(migrationfiles.Files, "00004_client_devices.sql")
	if err != nil {
		t.Fatalf("read client devices migration: %v", err)
	}
	failingMigrations := fstest.MapFS{
		"00001_bootstrap.sql":       {Data: bootstrap},
		"00002_users.sql":           {Data: users},
		"00003_bootstrap_state.sql": {Data: bootstrapState},
		"00004_client_devices.sql":  {Data: clientDevices},
		"00005_failure.sql": {
			Data: []byte("-- +goose Up\nSELECT * FROM table_that_does_not_exist;\n"),
		},
	}
	if err := migrate(ctx, database, failingMigrations); err == nil {
		t.Fatal("migrate() error = nil, want failing migration error")
	}
	assertMigrationState(t, ctx, database, 4, 4, false)
}

func assertMigrationState(t *testing.T, ctx context.Context, database *sql.DB, current, target int64, pending bool) {
	t.Helper()

	state, err := GetMigrationState(ctx, database)
	if err != nil {
		t.Fatalf("GetMigrationState() error = %v", err)
	}
	if state.CurrentVersion != current || state.TargetVersion != target || state.HasPending != pending {
		t.Fatalf(
			"migration state = %+v, want current %d, target %d, pending %t",
			state,
			current,
			target,
			pending,
		)
	}
}
