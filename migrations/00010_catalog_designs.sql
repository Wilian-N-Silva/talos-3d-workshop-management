-- +goose Up
CREATE TABLE catalog_parts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    catalog_item_id UUID NOT NULL,
    name TEXT NOT NULL
        CONSTRAINT catalog_parts_name_valid CHECK (char_length(btrim(name)) BETWEEN 1 AND 200),
    quantity INTEGER NOT NULL DEFAULT 1
        CONSTRAINT catalog_parts_quantity_valid CHECK (quantity > 0),
    notes TEXT NOT NULL DEFAULT ''
        CONSTRAINT catalog_parts_notes_valid CHECK (char_length(notes) <= 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT catalog_parts_item_fk FOREIGN KEY (catalog_item_id) REFERENCES catalog_items(id) ON DELETE CASCADE
);

CREATE INDEX catalog_parts_item_name_idx ON catalog_parts (catalog_item_id, name, id);

CREATE TABLE design_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    catalog_part_id UUID NOT NULL,
    version TEXT NOT NULL
        CONSTRAINT design_versions_version_valid CHECK (char_length(btrim(version)) BETWEEN 1 AND 100),
    notes TEXT NOT NULL DEFAULT ''
        CONSTRAINT design_versions_notes_valid CHECK (char_length(notes) <= 10000),
    origin TEXT NOT NULL DEFAULT 'unknown'
        CONSTRAINT design_versions_origin_valid CHECK (origin IN ('original', 'customer', 'remix', 'third_party', 'unknown')),
    source_url TEXT NULL
        CONSTRAINT design_versions_source_url_valid CHECK (source_url IS NULL OR char_length(source_url) <= 2048),
    original_author TEXT NOT NULL DEFAULT ''
        CONSTRAINT design_versions_author_valid CHECK (char_length(original_author) <= 200),
    license_name TEXT NOT NULL DEFAULT ''
        CONSTRAINT design_versions_license_valid CHECK (char_length(license_name) <= 200),
    commercial_use_allowed BOOLEAN NULL,
    attribution_required BOOLEAN NOT NULL DEFAULT FALSE,
    attribution_text TEXT NOT NULL DEFAULT ''
        CONSTRAINT design_versions_attribution_valid CHECK (char_length(attribution_text) <= 4000),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT design_versions_part_fk FOREIGN KEY (catalog_part_id) REFERENCES catalog_parts(id) ON DELETE RESTRICT,
    CONSTRAINT design_versions_part_version_unique UNIQUE (catalog_part_id, version)
);

CREATE INDEX design_versions_part_created_idx ON design_versions (catalog_part_id, created_at DESC, id);

CREATE TABLE design_version_files (
    design_version_id UUID NOT NULL REFERENCES design_versions(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE RESTRICT,
    role TEXT NOT NULL
        CONSTRAINT design_version_files_role_valid CHECK (role IN ('source', 'mesh', 'print', 'preview', 'documentation', 'other')),
    attached_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (design_version_id, file_id, role)
);

CREATE INDEX design_version_files_print_idx ON design_version_files (design_version_id, role) WHERE role = 'print';
