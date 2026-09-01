-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CONSTRAINT users_name_not_blank CHECK (btrim(name) <> ''),
    email_or_username TEXT NOT NULL
        CONSTRAINT users_email_or_username_not_blank CHECK (btrim(email_or_username) <> '')
        CONSTRAINT users_email_or_username_trimmed CHECK (email_or_username = btrim(email_or_username)),
    password_hash TEXT NOT NULL
        CONSTRAINT users_password_hash_not_blank CHECK (password_hash <> ''),
    status TEXT NOT NULL
        CONSTRAINT users_status_valid CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMPTZ,
    CONSTRAINT users_updated_at_not_before_created CHECK (updated_at >= created_at)
);

CREATE UNIQUE INDEX users_email_or_username_unique
    ON users (lower(email_or_username));
