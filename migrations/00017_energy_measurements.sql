-- +goose Up
CREATE TABLE energy_measurements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL CONSTRAINT energy_measurements_job_fk REFERENCES print_jobs(id) ON DELETE CASCADE,
    source VARCHAR(20) NOT NULL CONSTRAINT energy_measurements_source_valid CHECK (source IN ('manual_meter','smart_plug','estimated','imported')),
    meter_start_kwh NUMERIC(18,6) NULL CONSTRAINT energy_measurements_meter_start_valid CHECK (meter_start_kwh IS NULL OR meter_start_kwh >= 0),
    meter_end_kwh NUMERIC(18,6) NULL CONSTRAINT energy_measurements_meter_end_valid CHECK (meter_end_kwh IS NULL OR meter_end_kwh >= 0),
    measured_kwh NUMERIC(18,6) NULL CONSTRAINT energy_measurements_measured_valid CHECK (measured_kwh IS NULL OR measured_kwh >= 0),
    estimated_average_power_w NUMERIC(18,6) NULL CONSTRAINT energy_measurements_power_valid CHECK (estimated_average_power_w IS NULL OR estimated_average_power_w > 0),
    energy_rate_cents_per_kwh BIGINT NOT NULL CONSTRAINT energy_measurements_rate_valid CHECK (energy_rate_cents_per_kwh >= 0),
    occurred_at TIMESTAMPTZ NOT NULL,
    recorded_by UUID NOT NULL CONSTRAINT energy_measurements_recorded_by_fk REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT NOT NULL DEFAULT '' CONSTRAINT energy_measurements_notes_valid CHECK (char_length(notes) <= 10000),
    CONSTRAINT energy_measurements_meter_pair CHECK ((meter_start_kwh IS NULL) = (meter_end_kwh IS NULL)),
    CONSTRAINT energy_measurements_meter_order CHECK (meter_start_kwh IS NULL OR meter_end_kwh >= meter_start_kwh)
);

CREATE INDEX energy_measurements_job_time_idx ON energy_measurements (job_id, occurred_at DESC, id DESC);
