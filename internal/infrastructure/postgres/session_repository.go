package postgres

import (
	"context"
	"database/sql"
	"fmt"

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
	return session, nil
}
