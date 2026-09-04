package catalog

import (
	"errors"
	"time"
)

var (
	ErrBOMItemNotFound   = errors.New("catalog BOM item not found")
	ErrBOMSupplyConflict = errors.New("supply already exists in catalog BOM")
)

type BOMItem struct {
	ID              string
	CatalogItemID   string
	SupplyID        string
	QuantityPerUnit string
	WastePercent    string
	Notes           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type BOMValues struct {
	SupplyID        string
	QuantityPerUnit string
	WastePercent    string
	Notes           string
}

type BOMCostInput struct {
	Item                     BOMItem
	SupplyName               string
	SupplyUnit               string
	ReplacementUnitCostCents int64
}

type BOMPreviewLine struct {
	Item                             BOMItem
	SupplyName                       string
	SupplyUnit                       string
	ReplacementUnitCostCents         int64
	EffectiveQuantityPerUnit         string
	ExactReplacementCostCentsPerUnit string
}

type BOMPreview struct {
	Items                          []BOMPreviewLine
	ExactTotalReplacementCostCents string
	RoundingApplied                bool
}
