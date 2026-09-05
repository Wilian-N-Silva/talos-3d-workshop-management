package inventory

import (
	"errors"
	"time"
)

var (
	ErrMaterialNotFound       = errors.New("material not found")
	ErrSpoolNotFound          = errors.New("spool not found")
	ErrSpoolCodeConflict      = errors.New("spool code already exists")
	ErrInventoryHistoryExists = errors.New("inventory history prevents deletion")
	ErrMeasurementBelowTare   = errors.New("gross measurement is below spool tare")
)

type Material struct {
	ID                               string    `json:"id"`
	Manufacturer                     string    `json:"manufacturer"`
	Name                             string    `json:"name"`
	MaterialType                     string    `json:"material_type"`
	ColorName                        string    `json:"color_name"`
	ColorHex                         *string   `json:"color_hex"`
	NominalDensity                   string    `json:"nominal_density"`
	DefaultReplacementCostPerKgCents int64     `json:"default_replacement_cost_per_kg_cents"`
	Notes                            string    `json:"notes"`
	CreatedAt                        time.Time `json:"created_at"`
	UpdatedAt                        time.Time `json:"updated_at"`
}

type MaterialValues struct {
	Manufacturer                     string
	Name                             string
	MaterialType                     string
	ColorName                        string
	ColorHex                         *string
	NominalDensity                   string
	DefaultReplacementCostPerKgCents int64
	Notes                            string
}

type SpoolStatus string

const (
	SpoolSealed  SpoolStatus = "sealed"
	SpoolOpen    SpoolStatus = "open"
	SpoolStored  SpoolStatus = "stored"
	SpoolDrying  SpoolStatus = "drying"
	SpoolEmpty   SpoolStatus = "empty"
	SpoolRetired SpoolStatus = "retired"
)

type Spool struct {
	ID                        string      `json:"id"`
	Code                      string      `json:"code"`
	MaterialID                string      `json:"material_id"`
	NominalNetWeightG         string      `json:"nominal_net_weight_g"`
	TareWeightG               string      `json:"tare_weight_g"`
	GrossWeightAtOpenG        *string     `json:"gross_weight_at_open_g"`
	CurrentRemainingWeightG   *string     `json:"current_remaining_weight_g"`
	PurchaseCostCents         int64       `json:"purchase_cost_cents"`
	ReplacementCostPerKgCents int64       `json:"replacement_cost_per_kg_cents"`
	OpenedAt                  *time.Time  `json:"opened_at"`
	LastWeighedAt             *time.Time  `json:"last_weighed_at"`
	LastDriedAt               *time.Time  `json:"last_dried_at"`
	StorageLocation           string      `json:"storage_location"`
	StorageStatus             string      `json:"storage_status"`
	LotNumber                 string      `json:"lot_number"`
	Status                    SpoolStatus `json:"status"`
	CreatedAt                 time.Time   `json:"created_at"`
	UpdatedAt                 time.Time   `json:"updated_at"`
}

type SpoolValues struct {
	Code                      string
	MaterialID                string
	NominalNetWeightG         string
	TareWeightG               string
	GrossWeightAtOpenG        *string
	PurchaseCostCents         int64
	ReplacementCostPerKgCents int64
	OpenedAt                  *time.Time
	LastDriedAt               *time.Time
	StorageLocation           string
	StorageStatus             string
	LotNumber                 string
	Status                    SpoolStatus
}

type MeasurementSource string

const (
	MeasurementManual   MeasurementSource = "manual"
	MeasurementImported MeasurementSource = "imported"
	MeasurementOther    MeasurementSource = "other"
)

type SpoolMeasurement struct {
	ID                      string            `json:"id"`
	SpoolID                 string            `json:"spool_id"`
	MeasuredAt              time.Time         `json:"measured_at"`
	GrossWeightG            string            `json:"gross_weight_g"`
	DerivedRemainingWeightG string            `json:"derived_remaining_weight_g"`
	Source                  MeasurementSource `json:"source"`
	Notes                   string            `json:"notes"`
	RecordedBy              string            `json:"recorded_by"`
	CreatedAt               time.Time         `json:"created_at"`
}

type MeasurementValues struct {
	MeasuredAt   time.Time
	GrossWeightG string
	Source       MeasurementSource
	Notes        string
}
