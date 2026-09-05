-- +goose Up
CREATE TABLE materials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    manufacturer TEXT NOT NULL CONSTRAINT materials_manufacturer_valid CHECK (char_length(btrim(manufacturer)) BETWEEN 1 AND 200),
    name TEXT NOT NULL CONSTRAINT materials_name_valid CHECK (char_length(btrim(name)) BETWEEN 1 AND 200),
    material_type TEXT NOT NULL CONSTRAINT materials_type_valid CHECK (char_length(btrim(material_type)) BETWEEN 1 AND 100),
    color_name TEXT NOT NULL DEFAULT '' CONSTRAINT materials_color_name_valid CHECK (char_length(color_name) <= 100),
    color_hex TEXT NULL CONSTRAINT materials_color_hex_valid CHECK (color_hex IS NULL OR color_hex ~ '^#[0-9A-Fa-f]{6}$'),
    nominal_density NUMERIC(12,6) NOT NULL CONSTRAINT materials_density_valid CHECK (nominal_density > 0),
    default_replacement_cost_per_kg_cents BIGINT NOT NULL DEFAULT 0 CONSTRAINT materials_replacement_cost_valid CHECK (default_replacement_cost_per_kg_cents >= 0),
    notes TEXT NOT NULL DEFAULT '' CONSTRAINT materials_notes_valid CHECK (char_length(notes) <= 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX materials_name_idx ON materials (manufacturer, name, id);

CREATE TABLE material_spools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL CONSTRAINT material_spools_code_valid CHECK (char_length(btrim(code)) BETWEEN 1 AND 100),
    material_id UUID NOT NULL,
    nominal_net_weight_g NUMERIC(14,3) NOT NULL CONSTRAINT material_spools_nominal_weight_valid CHECK (nominal_net_weight_g > 0),
    tare_weight_g NUMERIC(14,3) NOT NULL CONSTRAINT material_spools_tare_weight_valid CHECK (tare_weight_g >= 0),
    gross_weight_at_open_g NUMERIC(14,3) NULL CONSTRAINT material_spools_open_weight_valid CHECK (gross_weight_at_open_g IS NULL OR gross_weight_at_open_g >= tare_weight_g),
    current_remaining_weight_g NUMERIC(14,3) NULL CONSTRAINT material_spools_remaining_weight_valid CHECK (current_remaining_weight_g IS NULL OR current_remaining_weight_g >= 0),
    purchase_cost_cents BIGINT NOT NULL DEFAULT 0 CONSTRAINT material_spools_purchase_cost_valid CHECK (purchase_cost_cents >= 0),
    replacement_cost_per_kg_cents BIGINT NOT NULL DEFAULT 0 CONSTRAINT material_spools_replacement_cost_valid CHECK (replacement_cost_per_kg_cents >= 0),
    opened_at TIMESTAMPTZ NULL,
    last_weighed_at TIMESTAMPTZ NULL,
    last_dried_at TIMESTAMPTZ NULL,
    storage_location TEXT NOT NULL DEFAULT '' CONSTRAINT material_spools_location_valid CHECK (char_length(storage_location) <= 200),
    storage_status TEXT NOT NULL DEFAULT '' CONSTRAINT material_spools_storage_status_valid CHECK (char_length(storage_status) <= 100),
    lot_number TEXT NOT NULL DEFAULT '' CONSTRAINT material_spools_lot_valid CHECK (char_length(lot_number) <= 200),
    status TEXT NOT NULL CONSTRAINT material_spools_status_valid CHECK (status IN ('sealed', 'open', 'stored', 'drying', 'empty', 'retired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT material_spools_material_fk FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX material_spools_code_unique ON material_spools (lower(code));
CREATE INDEX material_spools_material_status_idx ON material_spools (material_id, status, code);

CREATE TABLE spool_measurements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    spool_id UUID NOT NULL,
    measured_at TIMESTAMPTZ NOT NULL,
    gross_weight_g NUMERIC(14,3) NOT NULL CONSTRAINT spool_measurements_gross_valid CHECK (gross_weight_g >= 0),
    derived_remaining_weight_g NUMERIC(14,3) NOT NULL CONSTRAINT spool_measurements_remaining_valid CHECK (derived_remaining_weight_g >= 0),
    source TEXT NOT NULL CONSTRAINT spool_measurements_source_valid CHECK (source IN ('manual', 'imported', 'other')),
    notes TEXT NOT NULL DEFAULT '' CONSTRAINT spool_measurements_notes_valid CHECK (char_length(notes) <= 10000),
    recorded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT spool_measurements_spool_fk FOREIGN KEY (spool_id) REFERENCES material_spools(id) ON DELETE RESTRICT
);

CREATE INDEX spool_measurements_history_idx ON spool_measurements (spool_id, measured_at DESC, id DESC);
