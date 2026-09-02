-- +goose Up
ALTER TABLE users
    ADD COLUMN role TEXT NOT NULL DEFAULT 'viewer'
        CONSTRAINT users_role_valid CHECK (role IN ('owner', 'operator', 'designer', 'commercial', 'viewer'));

-- The bootstrap marker is the durable identity of the initial Owner. Existing
-- non-bootstrap users remain least-privilege Viewers during this upgrade.
UPDATE users AS u
SET role = 'owner'
FROM bootstrap_state AS b
WHERE u.id = b.initial_owner_user_id;
