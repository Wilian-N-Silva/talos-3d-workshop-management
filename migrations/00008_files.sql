-- +goose Up
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sha256 BYTEA NOT NULL
        CONSTRAINT files_sha256_length CHECK (octet_length(sha256) = 32),
    original_name TEXT NOT NULL
        CONSTRAINT files_original_name_valid CHECK (
            char_length(btrim(original_name)) BETWEEN 1 AND 255
        ),
    content_type TEXT NOT NULL
        CONSTRAINT files_content_type_valid CHECK (
            char_length(btrim(content_type)) BETWEEN 1 AND 200
        ),
    size_bytes BIGINT NOT NULL
        CONSTRAINT files_size_nonnegative CHECK (size_bytes >= 0),
    storage_key TEXT NOT NULL
        CONSTRAINT files_storage_key_valid CHECK (
            char_length(storage_key) BETWEEN 1 AND 128
            AND storage_key ~ '^[A-Za-z0-9_-]+$'
        ),
    uploaded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT files_sha256_unique UNIQUE (sha256),
    CONSTRAINT files_storage_key_unique UNIQUE (storage_key)
);

ALTER TABLE workshop_settings
    ADD CONSTRAINT workshop_settings_logo_file_fk
    FOREIGN KEY (logo_file_id) REFERENCES files(id) ON DELETE RESTRICT;
