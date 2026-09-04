-- +goose Up
CREATE TABLE print_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(100) NOT NULL CONSTRAINT print_jobs_code_valid CHECK (btrim(code) <> ''),
    catalog_item_id UUID NOT NULL CONSTRAINT print_jobs_catalog_item_fk REFERENCES catalog_items(id) ON DELETE RESTRICT,
    design_version_id UUID NOT NULL CONSTRAINT print_jobs_design_version_fk REFERENCES design_versions(id) ON DELETE RESTRICT,
    printer_id UUID NOT NULL CONSTRAINT print_jobs_printer_fk REFERENCES printers(id) ON DELETE RESTRICT,
    order_item_id UUID NULL,
    purpose VARCHAR(20) NOT NULL CONSTRAINT print_jobs_purpose_valid CHECK (purpose IN ('test','prototype','production','maintenance','internal','personal')),
    status VARCHAR(20) NOT NULL DEFAULT 'draft' CONSTRAINT print_jobs_status_valid CHECK (status IN ('draft','prepared','printing','awaiting_review','completed','failed','cancelled')),
    planned_quantity INTEGER NOT NULL CONSTRAINT print_jobs_planned_quantity_valid CHECK (planned_quantity > 0),
    good_quantity INTEGER NOT NULL DEFAULT 0 CONSTRAINT print_jobs_good_quantity_valid CHECK (good_quantity >= 0),
    scrap_quantity INTEGER NOT NULL DEFAULT 0 CONSTRAINT print_jobs_scrap_quantity_valid CHECK (scrap_quantity >= 0),
    hypothesis TEXT NOT NULL DEFAULT '' CONSTRAINT print_jobs_hypothesis_valid CHECK (char_length(hypothesis) <= 10000),
    result_notes TEXT NOT NULL DEFAULT '' CONSTRAINT print_jobs_result_notes_valid CHECK (char_length(result_notes) <= 10000),
    quality_status VARCHAR(20) NOT NULL DEFAULT 'pending' CONSTRAINT print_jobs_quality_status_valid CHECK (quality_status IN ('pending','approved','partial','failed')),
    planned_seconds BIGINT NOT NULL DEFAULT 0 CONSTRAINT print_jobs_planned_seconds_valid CHECK (planned_seconds >= 0),
    actual_seconds BIGINT NULL CONSTRAINT print_jobs_actual_seconds_valid CHECK (actual_seconds IS NULL OR actual_seconds >= 0),
    labor_minutes INTEGER NOT NULL DEFAULT 0 CONSTRAINT print_jobs_labor_minutes_valid CHECK (labor_minutes >= 0),
    created_by UUID NOT NULL CONSTRAINT print_jobs_created_by_fk REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX print_jobs_code_unique ON print_jobs (lower(code));
CREATE INDEX print_jobs_status_created_idx ON print_jobs (status, created_at DESC, id);
CREATE INDEX print_jobs_catalog_item_idx ON print_jobs (catalog_item_id, created_at DESC);
CREATE INDEX print_jobs_printer_idx ON print_jobs (printer_id, created_at DESC);

CREATE TABLE job_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL CONSTRAINT job_events_job_fk REFERENCES print_jobs(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL CONSTRAINT job_events_type_valid CHECK (event_type IN ('created','prepared','printing_started_manual','finished_manual','reviewed','failed','cancelled')),
    occurred_at TIMESTAMPTZ NOT NULL,
    actor_user_id UUID NOT NULL CONSTRAINT job_events_actor_fk REFERENCES users(id) ON DELETE RESTRICT,
    source_device_id UUID NOT NULL CONSTRAINT job_events_device_fk REFERENCES client_devices(id) ON DELETE RESTRICT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb CONSTRAINT job_events_metadata_object CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX job_events_job_time_idx ON job_events (job_id, occurred_at, id);
