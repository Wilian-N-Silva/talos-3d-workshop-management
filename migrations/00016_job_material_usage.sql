-- +goose Up
CREATE TABLE print_job_material_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    print_job_id UUID NOT NULL CONSTRAINT job_material_job_fk REFERENCES print_jobs(id) ON DELETE CASCADE,
    material_id UUID NOT NULL CONSTRAINT job_material_material_fk REFERENCES materials(id) ON DELETE RESTRICT,
    spool_id UUID NOT NULL CONSTRAINT job_material_spool_fk REFERENCES material_spools(id) ON DELETE RESTRICT,
    role VARCHAR(20) NOT NULL CONSTRAINT job_material_role_valid CHECK (role IN ('model','support','purge','other')),
    planned_grams NUMERIC(18,6) NOT NULL CONSTRAINT job_material_planned_grams_valid CHECK (planned_grams >= 0),
    actual_grams NUMERIC(18,6) NULL CONSTRAINT job_material_actual_grams_valid CHECK (actual_grams IS NULL OR actual_grams >= 0),
    planned_meters NUMERIC(18,6) NULL CONSTRAINT job_material_planned_meters_valid CHECK (planned_meters IS NULL OR planned_meters >= 0),
    actual_meters NUMERIC(18,6) NULL CONSTRAINT job_material_actual_meters_valid CHECK (actual_meters IS NULL OR actual_meters >= 0),
    measurement_source VARCHAR(30) NOT NULL CONSTRAINT job_material_source_valid CHECK (measurement_source IN ('slicer','spool_weight_delta','manual','printer','estimated')),
    historical_material_cost_cents BIGINT NULL CONSTRAINT job_material_historical_cost_valid CHECK (historical_material_cost_cents IS NULL OR historical_material_cost_cents >= 0),
    replacement_material_cost_cents BIGINT NULL CONSTRAINT job_material_replacement_cost_valid CHECK (replacement_material_cost_cents IS NULL OR replacement_material_cost_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT job_material_role_spool_unique UNIQUE (print_job_id, spool_id, role)
);

CREATE INDEX job_material_job_idx ON print_job_material_usage (print_job_id, role, id);
CREATE INDEX job_material_spool_idx ON print_job_material_usage (spool_id, id);
