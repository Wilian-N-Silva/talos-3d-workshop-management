package maintenance

import (
	"errors"
	"time"
)

var ErrMaintenanceReference = errors.New("invalid maintenance event reference")

type Type string

const (
	TypeCleaning    Type = "cleaning"
	TypePreventive  Type = "preventive"
	TypeCorrective  Type = "corrective"
	TypeReplacement Type = "replacement"
	TypeUpgrade     Type = "upgrade"
	TypeInspection  Type = "inspection"
)

type Event struct {
	ID              string    `json:"id"`
	PrinterID       string    `json:"printer_id"`
	Type            Type      `json:"type"`
	PerformedAt     time.Time `json:"performed_at"`
	PrinterHours    *string   `json:"printer_hours"`
	Description     string    `json:"description"`
	CostCents       *int64    `json:"cost_cents"`
	DowntimeMinutes int       `json:"downtime_minutes"`
	Notes           string    `json:"notes"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
}
type Values struct {
	Type            Type
	PerformedAt     time.Time
	PrinterHours    *string
	Description     string
	CostCents       *int64
	DowntimeMinutes int
	Notes           string
}
