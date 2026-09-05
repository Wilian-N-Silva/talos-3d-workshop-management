-- +goose Up
CREATE TABLE printers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL CONSTRAINT printers_name_valid CHECK (btrim(name) <> ''),
    manufacturer VARCHAR(200) NOT NULL CONSTRAINT printers_manufacturer_valid CHECK (btrim(manufacturer) <> ''),
    model VARCHAR(200) NOT NULL CONSTRAINT printers_model_valid CHECK (btrim(model) <> ''),
    nozzle_diameter NUMERIC(6,3) NOT NULL CONSTRAINT printers_nozzle_valid CHECK (nozzle_diameter > 0),
    location VARCHAR(500) NOT NULL DEFAULT '',
    acquisition_cost_cents BIGINT NOT NULL CONSTRAINT printers_acquisition_cost_valid CHECK (acquisition_cost_cents >= 0),
    residual_value_cents BIGINT NOT NULL CONSTRAINT printers_residual_value_valid CHECK (residual_value_cents >= 0 AND residual_value_cents <= acquisition_cost_cents),
    useful_life_hours NUMERIC(12,2) NOT NULL CONSTRAINT printers_useful_life_valid CHECK (useful_life_hours > 0),
    maintenance_reserve_per_hour_cents BIGINT NOT NULL CONSTRAINT printers_maintenance_reserve_valid CHECK (maintenance_reserve_per_hour_cents >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CONSTRAINT printers_status_valid CHECK (status IN ('active','maintenance','retired')),
    notes TEXT NOT NULL DEFAULT '' CONSTRAINT printers_notes_valid CHECK (char_length(notes) <= 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX printers_name_unique ON printers (lower(name));
CREATE INDEX printers_status_name_idx ON printers (status, name, id);
