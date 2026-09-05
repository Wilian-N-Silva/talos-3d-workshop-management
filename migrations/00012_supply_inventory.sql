-- +goose Up
CREATE TABLE supplies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL CONSTRAINT supplies_name_valid CHECK (char_length(btrim(name)) BETWEEN 1 AND 200),
    sku TEXT NULL CONSTRAINT supplies_sku_valid CHECK (sku IS NULL OR char_length(btrim(sku)) BETWEEN 1 AND 100),
    unit TEXT NOT NULL CONSTRAINT supplies_unit_valid CHECK (char_length(btrim(unit)) BETWEEN 1 AND 50),
    current_quantity NUMERIC(18,6) NOT NULL DEFAULT 0 CONSTRAINT supplies_current_quantity_valid CHECK (current_quantity >= 0),
    replacement_unit_cost_cents BIGINT NOT NULL DEFAULT 0 CONSTRAINT supplies_replacement_cost_valid CHECK (replacement_unit_cost_cents >= 0),
    minimum_quantity NUMERIC(18,6) NOT NULL DEFAULT 0 CONSTRAINT supplies_minimum_quantity_valid CHECK (minimum_quantity >= 0),
    notes TEXT NOT NULL DEFAULT '' CONSTRAINT supplies_notes_valid CHECK (char_length(notes) <= 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX supplies_sku_unique ON supplies (lower(sku)) WHERE sku IS NOT NULL;
CREATE INDEX supplies_name_idx ON supplies (name, id);
CREATE INDEX supplies_low_stock_idx ON supplies (current_quantity, minimum_quantity, id);

CREATE TABLE supply_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    supply_id UUID NOT NULL,
    type TEXT NOT NULL CONSTRAINT supply_movements_type_valid CHECK (type IN ('purchase', 'consume', 'adjustment', 'return', 'discard')),
    quantity NUMERIC(18,6) NOT NULL CONSTRAINT supply_movements_quantity_valid CHECK (
        (type IN ('purchase', 'return') AND quantity > 0)
        OR (type IN ('consume', 'discard') AND quantity < 0)
        OR (type = 'adjustment' AND quantity <> 0)
    ),
    unit_cost_cents BIGINT NULL CONSTRAINT supply_movements_unit_cost_valid CHECK (unit_cost_cents IS NULL OR unit_cost_cents >= 0),
    reference_type TEXT NULL,
    reference_id TEXT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    recorded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    notes TEXT NOT NULL DEFAULT '' CONSTRAINT supply_movements_notes_valid CHECK (char_length(notes) <= 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT supply_movements_supply_fk FOREIGN KEY (supply_id) REFERENCES supplies(id) ON DELETE RESTRICT,
    CONSTRAINT supply_movements_reference_pair CHECK ((reference_type IS NULL) = (reference_id IS NULL)),
    CONSTRAINT supply_movements_reference_type_valid CHECK (reference_type IS NULL OR char_length(btrim(reference_type)) BETWEEN 1 AND 100),
    CONSTRAINT supply_movements_reference_id_valid CHECK (reference_id IS NULL OR char_length(btrim(reference_id)) BETWEEN 1 AND 200)
);

CREATE INDEX supply_movements_history_idx ON supply_movements (supply_id, occurred_at DESC, id DESC);
