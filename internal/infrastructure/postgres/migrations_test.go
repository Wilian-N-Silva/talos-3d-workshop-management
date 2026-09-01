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

	if _, err := database.ExecContext(ctx, "DROP TABLE IF EXISTS goose_db_version"); err != nil {
		t.Fatalf("reset migration state: %v", err)
	}

	state, err := GetMigrationState(ctx, database)
	if err != nil {
		t.Fatalf("GetMigrationState() before migrate error = %v", err)
	}
	if state.CurrentVersion != 0 || state.TargetVersion != 1 || !state.HasPending {
		t.Fatalf("state before migrate = %+v, want current 0, target 1, pending", state)
	}

	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	assertMigrationState(t, ctx, database, 1, 1, false)

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
	failingMigrations := fstest.MapFS{
		"00001_bootstrap.sql": {Data: bootstrap},
		"00002_failure.sql": {
			Data: []byte("-- +goose Up\nSELECT * FROM table_that_does_not_exist;\n"),
		},
	}
	if err := migrate(ctx, database, failingMigrations); err == nil {
		t.Fatal("migrate() error = nil, want failing migration error")
	}
	assertMigrationState(t, ctx, database, 1, 1, false)
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
