-- +goose Up
CREATE TABLE catalog_bom_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    catalog_item_id UUID NOT NULL,
    supply_id UUID NOT NULL,
    quantity_per_unit NUMERIC(18,6) NOT NULL CONSTRAINT catalog_bom_quantity_valid CHECK (quantity_per_unit > 0),
    waste_percent NUMERIC(9,4) NOT NULL DEFAULT 0 CONSTRAINT catalog_bom_waste_valid CHECK (waste_percent >= 0),
    notes TEXT NOT NULL DEFAULT '' CONSTRAINT catalog_bom_notes_valid CHECK (char_length(notes) <= 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT catalog_bom_catalog_item_fk FOREIGN KEY (catalog_item_id) REFERENCES catalog_items(id) ON DELETE CASCADE,
    CONSTRAINT catalog_bom_supply_fk FOREIGN KEY (supply_id) REFERENCES supplies(id) ON DELETE RESTRICT,
    CONSTRAINT catalog_bom_item_supply_unique UNIQUE (catalog_item_id, supply_id)
);

CREATE INDEX catalog_bom_item_idx ON catalog_bom_items (catalog_item_id, id);
CREATE INDEX catalog_bom_supply_idx ON catalog_bom_items (supply_id, id);
