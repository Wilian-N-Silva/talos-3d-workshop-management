package printers

import (
	"errors"
	"time"
)

var (
	ErrPrinterNotFound     = errors.New("printer not found")
	ErrPrinterNameConflict = errors.New("printer name already exists")
)

type Status string

const (
	StatusActive      Status = "active"
	StatusMaintenance Status = "maintenance"
	StatusRetired     Status = "retired"
)

type Printer struct {
	ID                             string    `json:"id"`
	Name                           string    `json:"name"`
	Manufacturer                   string    `json:"manufacturer"`
	Model                          string    `json:"model"`
	NozzleDiameter                 string    `json:"nozzle_diameter"`
	Location                       string    `json:"location"`
	AcquisitionCostCents           int64     `json:"acquisition_cost_cents"`
	ResidualValueCents             int64     `json:"residual_value_cents"`
	UsefulLifeHours                string    `json:"useful_life_hours"`
	MaintenanceReservePerHourCents int64     `json:"maintenance_reserve_per_hour_cents"`
	Status                         Status    `json:"status"`
	Notes                          string    `json:"notes"`
	CreatedAt                      time.Time `json:"created_at"`
	UpdatedAt                      time.Time `json:"updated_at"`
}

type Values struct {
	Name                           string
	Manufacturer                   string
	Model                          string
	NozzleDiameter                 string
	Location                       string
	AcquisitionCostCents           int64
	ResidualValueCents             int64
	UsefulLifeHours                string
	MaintenanceReservePerHourCents int64
	Status                         Status
	Notes                          string
}
