-- +goose Up
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    device_id UUID NOT NULL REFERENCES client_devices(id) ON DELETE RESTRICT,
    token_hash BYTEA NOT NULL
        CONSTRAINT sessions_token_hash_sha256_length CHECK (octet_length(token_hash) = 32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT sessions_token_hash_unique UNIQUE (token_hash),
    CONSTRAINT sessions_expiry_after_creation CHECK (expires_at > created_at),
    CONSTRAINT sessions_last_used_not_before_creation CHECK (last_used_at IS NULL OR last_used_at >= created_at),
    CONSTRAINT sessions_revoked_not_before_creation CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);

CREATE INDEX sessions_user_id_index ON sessions (user_id);
CREATE INDEX sessions_device_id_index ON sessions (device_id);
