-- +goose Up
CREATE TABLE labor_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CONSTRAINT labor_rates_name_valid CHECK (char_length(btrim(name)) BETWEEN 1 AND 200),
    activity_type VARCHAR(30) NOT NULL CONSTRAINT labor_rates_activity_valid CHECK (activity_type IN ('setup','material_handling','support_removal','finishing','painting','assembly','packaging','modeling','customization','consulting','other')),
    cost_hourly_rate_cents BIGINT NOT NULL CONSTRAINT labor_rates_cost_valid CHECK (cost_hourly_rate_cents >= 0),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX labor_rates_name_unique ON labor_rates (lower(name));
CREATE INDEX labor_rates_active_activity_idx ON labor_rates (active, activity_type, name);

CREATE TABLE job_labor_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL CONSTRAINT job_labor_entries_job_fk REFERENCES print_jobs(id) ON DELETE CASCADE,
    labor_rate_id UUID NOT NULL CONSTRAINT job_labor_entries_rate_fk REFERENCES labor_rates(id) ON DELETE RESTRICT,
    activity_type VARCHAR(30) NOT NULL CONSTRAINT job_labor_entries_activity_valid CHECK (activity_type IN ('setup','material_handling','support_removal','finishing','painting','assembly','packaging','modeling','customization','consulting','other')),
    minutes INTEGER NOT NULL CONSTRAINT job_labor_entries_minutes_valid CHECK (minutes > 0),
    internal_hourly_rate_cents BIGINT NOT NULL CONSTRAINT job_labor_entries_rate_snapshot_valid CHECK (internal_hourly_rate_cents >= 0),
    occurred_at TIMESTAMPTZ NOT NULL,
    recorded_by UUID NOT NULL CONSTRAINT job_labor_entries_recorded_by_fk REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT NOT NULL DEFAULT '' CONSTRAINT job_labor_entries_notes_valid CHECK (char_length(notes) <= 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX job_labor_entries_job_time_idx ON job_labor_entries (job_id, occurred_at DESC, id DESC);
