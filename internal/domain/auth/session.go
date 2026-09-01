package auth

import "time"

// Session is a server-side bearer-session record. TokenHash contains only the
// SHA-256 digest of the opaque token issued to the desktop client.
type Session struct {
	ID         string
	UserID     string
	DeviceID   string
	TokenHash  []byte
	CreatedAt  time.Time
	ExpiresAt  time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

// CreateSessionParams contains hash-only session data ready for persistence.
type CreateSessionParams struct {
	UserID    string
	DeviceID  string
	TokenHash []byte
	ExpiresAt time.Time
}
