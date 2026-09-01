-- +goose Up
CREATE TABLE bootstrap_state (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE
        CONSTRAINT bootstrap_state_singleton_true CHECK (singleton),
    initial_owner_user_id UUID NOT NULL UNIQUE
        REFERENCES users(id) ON DELETE RESTRICT,
    completed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Databases that created a user before this migration are already closed for
-- bootstrap. Preserve that state by recording the oldest user as the initial
-- owner marker.
INSERT INTO bootstrap_state (singleton, initial_owner_user_id)
SELECT TRUE, id
FROM users
ORDER BY created_at, id
LIMIT 1;
