package inventory

import (
	"errors"
	"time"
)

var (
	ErrSupplyNotFound      = errors.New("supply not found")
	ErrSupplySKUConflict   = errors.New("supply SKU already exists")
	ErrSupplyHistoryExists = errors.New("supply movement history prevents deletion")
	ErrInsufficientStock   = errors.New("supply movement would make stock negative")
)

type Supply struct {
	ID                       string    `json:"id"`
	Name                     string    `json:"name"`
	SKU                      *string   `json:"sku"`
	Unit                     string    `json:"unit"`
	CurrentQuantity          string    `json:"current_quantity"`
	ReplacementUnitCostCents int64     `json:"replacement_unit_cost_cents"`
	MinimumQuantity          string    `json:"minimum_quantity"`
	Notes                    string    `json:"notes"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type SupplyValues struct {
	Name                     string
	SKU                      *string
	Unit                     string
	ReplacementUnitCostCents int64
	MinimumQuantity          string
	Notes                    string
}

type SupplyMovementType string

const (
	SupplyPurchase   SupplyMovementType = "purchase"
	SupplyConsume    SupplyMovementType = "consume"
	SupplyAdjustment SupplyMovementType = "adjustment"
	SupplyReturn     SupplyMovementType = "return"
	SupplyDiscard    SupplyMovementType = "discard"
)

type SupplyMovement struct {
	ID            string             `json:"id"`
	SupplyID      string             `json:"supply_id"`
	Type          SupplyMovementType `json:"type"`
	Quantity      string             `json:"quantity"`
	UnitCostCents *int64             `json:"unit_cost_cents"`
	ReferenceType *string            `json:"reference_type"`
	ReferenceID   *string            `json:"reference_id"`
	OccurredAt    time.Time          `json:"occurred_at"`
	RecordedBy    string             `json:"recorded_by"`
	Notes         string             `json:"notes"`
	CreatedAt     time.Time          `json:"created_at"`
}

type SupplyMovementValues struct {
	Type          SupplyMovementType
	Quantity      string
	UnitCostCents *int64
	ReferenceType *string
	ReferenceID   *string
	OccurredAt    time.Time
	Notes         string
}

type LowInventory struct {
	SpoolThresholdG string   `json:"spool_threshold_g"`
	Spools          []Spool  `json:"spools"`
	Supplies        []Supply `json:"supplies"`
}
