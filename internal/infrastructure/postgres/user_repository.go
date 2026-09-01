package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

// CreateUserParams contains the values required to persist a new user.
type CreateUserParams struct {
	Name            string
	EmailOrUsername string
	PasswordHash    string
	Status          auth.UserStatus
}

// UserRepository persists authentication users in PostgreSQL.
type UserRepository struct {
	database *sql.DB
}

// NewUserRepository creates a PostgreSQL user repository.
func NewUserRepository(database *sql.DB) *UserRepository {
	return &UserRepository{database: database}
}

// Create inserts a user and returns database-generated identity and timestamps.
func (repository *UserRepository) Create(ctx context.Context, params CreateUserParams) (auth.User, error) {
	row := repository.database.QueryRowContext(
		ctx,
		`INSERT INTO users (name, email_or_username, password_hash, status)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, email_or_username, password_hash, status,
		           created_at, updated_at, last_login_at`,
		params.Name,
		params.EmailOrUsername,
		params.PasswordHash,
		params.Status,
	)

	user, err := scanUser(row)
	if err != nil {
		return auth.User{}, fmt.Errorf("create user: %w", err)
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
