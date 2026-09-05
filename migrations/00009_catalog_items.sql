-- +goose Up
CREATE TABLE catalog_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL
        CONSTRAINT catalog_items_name_valid CHECK (char_length(btrim(name)) BETWEEN 1 AND 200),
    sku TEXT NULL
        CONSTRAINT catalog_items_sku_valid CHECK (sku IS NULL OR char_length(btrim(sku)) BETWEEN 1 AND 100),
    description TEXT NOT NULL DEFAULT ''
        CONSTRAINT catalog_items_description_valid CHECK (char_length(description) <= 10000),
    purpose TEXT NOT NULL
        CONSTRAINT catalog_items_purpose_valid CHECK (
            purpose IN ('product', 'prototype', 'tooling', 'test', 'internal', 'personal')
        ),
    sellable BOOLEAN NOT NULL DEFAULT FALSE,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb
        CONSTRAINT catalog_items_tags_array CHECK (jsonb_typeof(tags) = 'array'),
    status TEXT NOT NULL DEFAULT 'active'
        CONSTRAINT catalog_items_status_valid CHECK (status IN ('active', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX catalog_items_name_id_idx ON catalog_items (name, id);
CREATE INDEX catalog_items_purpose_status_idx ON catalog_items (purpose, status);
CREATE INDEX catalog_items_tags_gin_idx ON catalog_items USING GIN (tags);
