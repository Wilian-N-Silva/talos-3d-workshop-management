package jobs

import (
	"errors"
	"time"
)

var (
	ErrMaterialUsageNotFound = errors.New("job material usage not found")
	ErrMaterialUsageConflict = errors.New("spool and role already exist on job")
)

type MaterialRole string

const (
	MaterialRoleModel   MaterialRole = "model"
	MaterialRoleSupport MaterialRole = "support"
	MaterialRolePurge   MaterialRole = "purge"
	MaterialRoleOther   MaterialRole = "other"
)

type MeasurementSource string

const (
	SourceSlicer           MeasurementSource = "slicer"
	SourceSpoolWeightDelta MeasurementSource = "spool_weight_delta"
	SourceManual           MeasurementSource = "manual"
	SourcePrinter          MeasurementSource = "printer"
	SourceEstimated        MeasurementSource = "estimated"
)

type MaterialUsage struct {
	ID                           string            `json:"id"`
	PrintJobID                   string            `json:"print_job_id"`
	MaterialID                   string            `json:"material_id"`
	SpoolID                      string            `json:"spool_id"`
	Role                         MaterialRole      `json:"role"`
	PlannedGrams                 string            `json:"planned_grams"`
	ActualGrams                  *string           `json:"actual_grams"`
	PlannedMeters                *string           `json:"planned_meters"`
	ActualMeters                 *string           `json:"actual_meters"`
	MeasurementSource            MeasurementSource `json:"measurement_source"`
	HistoricalMaterialCostCents  *int64            `json:"historical_material_cost_cents"`
	ReplacementMaterialCostCents *int64            `json:"replacement_material_cost_cents"`
	CreatedAt                    time.Time         `json:"created_at"`
	UpdatedAt                    time.Time         `json:"updated_at"`
}
type MaterialUsageValues struct {
	SpoolID                      string
	Role                         MaterialRole
	PlannedGrams                 string
	ActualGrams                  *string
	PlannedMeters                *string
	ActualMeters                 *string
	MeasurementSource            MeasurementSource
	HistoricalMaterialCostCents  *int64
	ReplacementMaterialCostCents *int64
}
type MaterialUsageSummary struct {
	Items             []MaterialUsage `json:"items"`
	TotalPlannedGrams string          `json:"total_planned_grams"`
	TotalActualGrams  string          `json:"total_actual_grams"`
}
