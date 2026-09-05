-- +goose Up
CREATE TABLE maintenance_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    printer_id UUID NOT NULL CONSTRAINT maintenance_events_printer_fk REFERENCES printers(id) ON DELETE RESTRICT,
    type VARCHAR(20) NOT NULL CONSTRAINT maintenance_events_type_valid CHECK (type IN ('cleaning','preventive','corrective','replacement','upgrade','inspection')),
    performed_at TIMESTAMPTZ NOT NULL,
    printer_hours NUMERIC(18,3) NULL CONSTRAINT maintenance_events_printer_hours_valid CHECK (printer_hours IS NULL OR printer_hours >= 0),
    description TEXT NOT NULL CONSTRAINT maintenance_events_description_valid CHECK (char_length(btrim(description)) BETWEEN 1 AND 10000),
    cost_cents BIGINT NULL CONSTRAINT maintenance_events_cost_valid CHECK (cost_cents IS NULL OR cost_cents >= 0),
    downtime_minutes INTEGER NOT NULL DEFAULT 0 CONSTRAINT maintenance_events_downtime_valid CHECK (downtime_minutes >= 0),
    notes TEXT NOT NULL DEFAULT '' CONSTRAINT maintenance_events_notes_valid CHECK (char_length(notes) <= 10000),
    created_by UUID NOT NULL CONSTRAINT maintenance_events_created_by_fk REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX maintenance_events_printer_time_idx ON maintenance_events (printer_id, performed_at DESC, id DESC);
