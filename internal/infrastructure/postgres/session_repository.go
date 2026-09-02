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

// FindByID resolves a session for ownership checks before revocation.
func (repository *SessionRepository) FindByID(ctx context.Context, id string) (auth.Session, error) {
	session, err := scanSession(repository.database.QueryRowContext(
		ctx,
		`SELECT `+sessionColumns+` FROM sessions WHERE id = $1`,
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	if err != nil {
		return auth.Session{}, fmt.Errorf("find session by ID: %w", err)
	}
	return session, nil
}

// ListByUserID returns newest-first session and device audit metadata.
func (repository *SessionRepository) ListByUserID(
	ctx context.Context,
	userID string,
) ([]auth.SessionDetails, error) {
	rows, err := repository.database.QueryContext(
		ctx,
		`SELECT s.id, s.user_id, s.device_id, s.created_at,
		        s.expires_at, s.last_used_at, s.revoked_at,
		        d.id, d.display_name, d.os, d.app_version, d.created_at, d.last_seen_at
		 FROM sessions AS s
		 JOIN client_devices AS d ON d.id = s.device_id
		 WHERE s.user_id = $1
		 ORDER BY s.created_at DESC, s.id DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions by user ID: %w", err)
	}
	defer rows.Close()

	details := make([]auth.SessionDetails, 0)
	for rows.Next() {
		detail, err := scanSessionDetails(rows)
		if err != nil {
			return nil, fmt.Errorf("scan session list: %w", err)
		}
		details = append(details, detail)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate session list: %w", err)
	}
	return details, nil
}

// Revoke sets revoked_at once and returns the persisted session. Repeating the
// operation preserves the original audit timestamp.
func (repository *SessionRepository) Revoke(
	ctx context.Context,
	id string,
	revokedAt time.Time,
) (auth.Session, error) {
	session, err := scanSession(repository.database.QueryRowContext(
		ctx,
		`UPDATE sessions
		 SET revoked_at = COALESCE(revoked_at, $2)
		 WHERE id = $1
		 RETURNING `+sessionColumns,
		id,
		revokedAt.UTC(),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return auth.Session{}, auth.ErrSessionNotFound
	}
	if err != nil {
		return auth.Session{}, fmt.Errorf("revoke session: %w", err)
	}
	return session, nil
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

func scanSessionDetails(row rowScanner) (auth.SessionDetails, error) {
	var details auth.SessionDetails
	var lastUsedAt sql.NullTime
	var revokedAt sql.NullTime
	if err := row.Scan(
		&details.Session.ID,
		&details.Session.UserID,
		&details.Session.DeviceID,
		&details.Session.CreatedAt,
		&details.Session.ExpiresAt,
		&lastUsedAt,
		&revokedAt,
		&details.Device.ID,
		&details.Device.DisplayName,
		&details.Device.OS,
		&details.Device.AppVersion,
		&details.Device.CreatedAt,
		&details.Device.LastSeenAt,
	); err != nil {
		return auth.SessionDetails{}, err
	}
	normalizeSession(&details.Session, lastUsedAt, revokedAt)
	details.Device.CreatedAt = details.Device.CreatedAt.UTC()
	details.Device.LastSeenAt = details.Device.LastSeenAt.UTC()
	return details, nil
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
