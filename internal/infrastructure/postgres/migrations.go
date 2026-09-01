package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"time"

	migrationfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/migrations"
	"github.com/pressly/goose/v3"
)

const migrationLockID int64 = 0x54414c4f533344

// MigrationState describes the database and embedded migration versions.
type MigrationState struct {
	CurrentVersion int64
	TargetVersion  int64
	HasPending     bool
}

// IsCurrent reports whether the database matches the embedded migration set.
func (state MigrationState) IsCurrent() bool {
	return !state.HasPending && state.CurrentVersion == state.TargetVersion
}

// Migrate applies all pending embedded migrations under a PostgreSQL advisory
// lock. Failure prevents server startup.
func Migrate(ctx context.Context, database *sql.DB) error {
	return migrate(ctx, database, migrationfiles.Files)
}

// GetMigrationState reports whether the database is at the embedded target.
func GetMigrationState(ctx context.Context, database *sql.DB) (MigrationState, error) {
	provider, err := goose.NewProvider(goose.DialectPostgres, database, migrationfiles.Files)
	if err != nil {
		return MigrationState{}, fmt.Errorf("create migration provider: %w", err)
	}

	current, target, err := provider.GetVersions(ctx)
	if err != nil {
		return MigrationState{}, fmt.Errorf("read migration versions: %w", err)
	}
	pending, err := provider.HasPending(ctx)
	if err != nil {
		return MigrationState{}, fmt.Errorf("read pending migrations: %w", err)
	}

	return MigrationState{
		CurrentVersion: current,
		TargetVersion:  target,
		HasPending:     pending,
	}, nil
}

func migrate(ctx context.Context, database *sql.DB, migrationFS fs.FS) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, database, migrationFS)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	return withMigrationLock(ctx, database, func() error {
		if _, err := provider.Up(ctx); err != nil {
			return fmt.Errorf("apply database migrations: %w", err)
		}
		return nil
	})
}

func withMigrationLock(ctx context.Context, database *sql.DB, action func() error) (returnErr error) {
	connection, err := database.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve migration lock connection: %w", err)
	}
	defer connection.Close()

	if _, err := connection.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrationLockID); err != nil {
		return fmt.Errorf("acquire migration advisory lock: %w", err)
	}

	defer func() {
		unlockContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := connection.ExecContext(unlockContext, "SELECT pg_advisory_unlock($1)", migrationLockID); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("release migration advisory lock: %w", err))
		}
	}()

	return action()
}
