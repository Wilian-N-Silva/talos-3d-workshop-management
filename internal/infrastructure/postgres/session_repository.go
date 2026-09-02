package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
)

const sessionColumns = `id, user_id, device_id, token_hash, created_at, expires_at, last_used_at, revoked_at`

// SessionRepository persists hash-only desktop bearer sessions in PostgreSQL.
type SessionRepository struct {
	database *sql.DB
}

// NewSessionRepository creates a PostgreSQL session repository.
func NewSessionRepository(database *sql.DB) *SessionRepository {
	return &SessionRepository{database: database}
}

// Create inserts a session without ever receiving its plaintext bearer token.
func (repository *SessionRepository) Create(
	ctx context.Context,
	params auth.CreateSessionParams,
) (auth.Session, error) {
	session, err := scanSession(repository.database.QueryRowContext(
		ctx,
		`INSERT INTO sessions (user_id, device_id, token_hash, expires_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING `+sessionColumns,
		params.UserID,
		params.DeviceID,
		params.TokenHash,
		params.ExpiresAt.UTC(),
	))
	if err != nil {
		return auth.Session{}, fmt.Errorf("create session: %w", err)
	}
	return session, nil
}

// FindByTokenHash resolves a session and its user without accepting the
// plaintext bearer token at the persistence boundary.
func (repository *SessionRepository) FindByTokenHash(
	ctx context.Context,
	tokenHash []byte,
) (auth.Session, auth.User, error) {
	row := repository.database.QueryRowContext(
		ctx,
		`SELECT s.id, s.user_id, s.device_id, s.token_hash, s.created_at,
		        s.expires_at, s.last_used_at, s.revoked_at,
		        u.id, u.name, u.email_or_username, u.password_hash, u.status, u.role,
		        u.created_at, u.updated_at, u.last_login_at
		 FROM sessions AS s
		 JOIN users AS u ON u.id = s.user_id
		 WHERE s.token_hash = $1`,
		tokenHash,
	)

	session, user, err := scanSessionAndUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Session{}, auth.User{}, auth.ErrSessionNotFound
	}
	if err != nil {
		return auth.Session{}, auth.User{}, fmt.Errorf("find session by token hash: %w", err)
	}
	return session, user, nil
}

// UpdateLastUsed advances last-used only when the stored value is old enough.
// The conditional update ensures concurrent requests produce at most one write.
func (repository *SessionRepository) UpdateLastUsed(
	ctx context.Context,
	id string,
	observedAt time.Time,
	updateBefore time.Time,
) (bool, error) {
	result, err := repository.database.ExecContext(
		ctx,
		`UPDATE sessions
		 SET last_used_at = GREATEST(COALESCE(last_used_at, created_at), $2)
		 WHERE id = $1
		   AND (last_used_at IS NULL OR last_used_at <= $3)`,
		id,
		observedAt.UTC(),
		updateBefore.UTC(),
	)
	if err != nil {
		return false, fmt.Errorf("update session last used: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read session last-used update result: %w", err)
	}
	return rowsAffected == 1, nil
}

func scanSession(row rowScanner) (auth.Session, error) {
	var session auth.Session
	var lastUsedAt sql.NullTime
	var revokedAt sql.NullTime
	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.DeviceID,
		&session.TokenHash,
		&session.CreatedAt,
		&session.ExpiresAt,
		&lastUsedAt,
		&revokedAt,
	); err != nil {
		return auth.Session{}, err
	}
	normalizeSession(&session, lastUsedAt, revokedAt)
	return session, nil
}

func normalizeSession(session *auth.Session, lastUsedAt, revokedAt sql.NullTime) {
	session.TokenHash = append([]byte(nil), session.TokenHash...)
	session.CreatedAt = session.CreatedAt.UTC()
	session.ExpiresAt = session.ExpiresAt.UTC()
	if lastUsedAt.Valid {
		value := lastUsedAt.Time.UTC()
		session.LastUsedAt = &value
	}
	if revokedAt.Valid {
		value := revokedAt.Time.UTC()
		session.RevokedAt = &value
	}
}

func scanSessionAndUser(row rowScanner) (auth.Session, auth.User, error) {
	var session auth.Session
	var user auth.User
	var lastUsedAt sql.NullTime
	var revokedAt sql.NullTime
	var lastLoginAt sql.NullTime
	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.DeviceID,
		&session.TokenHash,
		&session.CreatedAt,
		&session.ExpiresAt,
		&lastUsedAt,
		&revokedAt,
		&user.ID,
		&user.Name,
		&user.EmailOrUsername,
		&user.PasswordHash,
		&user.Status,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLoginAt,
	); err != nil {
		return auth.Session{}, auth.User{}, err
	}

	normalizeSession(&session, lastUsedAt, revokedAt)
	user.CreatedAt = user.CreatedAt.UTC()
	user.UpdatedAt = user.UpdatedAt.UTC()
	if lastLoginAt.Valid {
		value := lastLoginAt.Time.UTC()
		user.LastLoginAt = &value
	}
	return session, user, nil
}
