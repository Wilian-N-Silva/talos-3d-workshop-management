package energy

import (
	"errors"
	"time"
)

var ErrMeasurementNotFound = errors.New("energy measurement not found")

type Source string

const (
	SourceManualMeter Source = "manual_meter"
	SourceSmartPlug   Source = "smart_plug"
	SourceEstimated   Source = "estimated"
	SourceImported    Source = "imported"
)

type Measurement struct {
	ID                     string    `json:"id"`
	JobID                  string    `json:"job_id"`
	Source                 Source    `json:"source"`
	MeterStartKWh          *string   `json:"meter_start_kwh"`
	MeterEndKWh            *string   `json:"meter_end_kwh"`
	MeasuredKWh            *string   `json:"measured_kwh"`
	EstimatedAveragePowerW *string   `json:"estimated_average_power_w"`
	EnergyRateCentsPerKWh  int64     `json:"energy_rate_cents_per_kwh"`
	OccurredAt             time.Time `json:"occurred_at"`
	RecordedBy             string    `json:"recorded_by"`
	Notes                  string    `json:"notes"`
}

type Values struct {
	Source                 Source
	MeterStartKWh          *string
	MeterEndKWh            *string
	MeasuredKWh            *string
	EstimatedAveragePowerW *string
	EnergyRateCentsPerKWh  int64
	OccurredAt             time.Time
	Notes                  string
}
