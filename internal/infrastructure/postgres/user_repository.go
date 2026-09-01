package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const firstUserLockID int64 = 0x54414c4f53555352

// UserRepository persists authentication users in PostgreSQL.
type UserRepository struct {
	database *sql.DB
}

// NewUserRepository creates a PostgreSQL user repository.
func NewUserRepository(database *sql.DB) *UserRepository {
	return &UserRepository{database: database}
}

// Create inserts a user and returns database-generated identity and timestamps.
func (repository *UserRepository) Create(ctx context.Context, params auth.CreateUserParams) (auth.User, error) {
	user, err := insertUser(ctx, repository.database, params)
	if err != nil {
		return auth.User{}, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// NeedsSetup reports whether first-user bootstrap has never completed and no
// user exists. Any existing user closes bootstrap even if the marker is absent.
func (repository *UserRepository) NeedsSetup(ctx context.Context) (bool, error) {
	var needsSetup bool
	err := repository.database.QueryRowContext(
		ctx,
		`SELECT NOT EXISTS (SELECT 1 FROM bootstrap_state)
		        AND NOT EXISTS (SELECT 1 FROM users)`,
	).Scan(&needsSetup)
	if err != nil {
		return false, fmt.Errorf("read bootstrap state: %w", err)
	}
	return needsSetup, nil
}

// CreateFirst atomically creates the initial owner and permanently closes
// bootstrap across concurrent requests and server instances.
func (repository *UserRepository) CreateFirst(
	ctx context.Context,
	params auth.CreateUserParams,
) (user auth.User, returnErr error) {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return auth.User{}, fmt.Errorf("begin first-user transaction: %w", err)
	}
	defer func() {
		if err := transaction.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			returnErr = errors.Join(returnErr, fmt.Errorf("rollback first-user transaction: %w", err))
		}
	}()

	if _, err := transaction.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", firstUserLockID); err != nil {
		return auth.User{}, fmt.Errorf("lock first-user creation: %w", err)
	}

	var closed bool
	if err := transaction.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM bootstrap_state)
		        OR EXISTS (SELECT 1 FROM users)`,
	).Scan(&closed); err != nil {
		return auth.User{}, fmt.Errorf("check first-user state: %w", err)
	}
	if closed {
		return auth.User{}, auth.ErrFirstUserAlreadyExists
	}

	user, err = insertUser(ctx, transaction, params)
	if err != nil {
		return auth.User{}, fmt.Errorf("create first user: %w", err)
	}
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO bootstrap_state (initial_owner_user_id) VALUES ($1)`,
		user.ID,
	); err != nil {
		return auth.User{}, fmt.Errorf("close first-user bootstrap: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return auth.User{}, fmt.Errorf("commit first-user transaction: %w", err)
	}
	return user, nil
}

// FindByEmailOrUsername finds a user using the login identifier's
// case-insensitive uniqueness rules.
func (repository *UserRepository) FindByEmailOrUsername(
	ctx context.Context,
	emailOrUsername string,
) (auth.User, error) {
	row := repository.database.QueryRowContext(
		ctx,
		`SELECT id, name, email_or_username, password_hash, status,
		        created_at, updated_at, last_login_at
		 FROM users
		 WHERE lower(email_or_username) = lower($1)`,
		emailOrUsername,
	)

	user, err := scanUser(row)
	if err != nil {
		return auth.User{}, fmt.Errorf("find user by login identifier: %w", err)
	}
	return user, nil
}

// Count returns the number of persisted users.
func (repository *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := repository.database.QueryRowContext(ctx, `SELECT count(*) FROM users`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func insertUser(ctx context.Context, database queryRower, params auth.CreateUserParams) (auth.User, error) {
	return scanUser(database.QueryRowContext(
		ctx,
		`INSERT INTO users (name, email_or_username, password_hash, status)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, email_or_username, password_hash, status,
		           created_at, updated_at, last_login_at`,
		params.Name,
		params.EmailOrUsername,
		params.PasswordHash,
		params.Status,
	))
}

func scanUser(row rowScanner) (auth.User, error) {
	var user auth.User
	var lastLoginAt sql.NullTime
	if err := row.Scan(
		&user.ID,
		&user.Name,
		&user.EmailOrUsername,
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
	); err != nil {
		return auth.User{}, err
	}

	user.CreatedAt = user.CreatedAt.UTC()
	user.UpdatedAt = user.UpdatedAt.UTC()
	if lastLoginAt.Valid {
		value := lastLoginAt.Time.UTC()
		user.LastLoginAt = &value
	}
	return user, nil
}
