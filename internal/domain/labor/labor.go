package labor

import (
	"errors"
	"time"
)

var (
	ErrRateNotFound = errors.New("labor rate not found")
	ErrRateConflict = errors.New("labor rate name already exists")
)

type ActivityType string

const (
	ActivitySetup            ActivityType = "setup"
	ActivityMaterialHandling ActivityType = "material_handling"
	ActivitySupportRemoval   ActivityType = "support_removal"
	ActivityFinishing        ActivityType = "finishing"
	ActivityPainting         ActivityType = "painting"
	ActivityAssembly         ActivityType = "assembly"
	ActivityPackaging        ActivityType = "packaging"
	ActivityModeling         ActivityType = "modeling"
	ActivityCustomization    ActivityType = "customization"
	ActivityConsulting       ActivityType = "consulting"
	ActivityOther            ActivityType = "other"
)

type Rate struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	ActivityType        ActivityType `json:"activity_type"`
	CostHourlyRateCents int64        `json:"cost_hourly_rate_cents"`
	Active              bool         `json:"active"`
	CreatedAt           time.Time    `json:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}
type RateValues struct {
	Name                string
	ActivityType        ActivityType
	CostHourlyRateCents int64
	Active              bool
}
type Entry struct {
	ID                      string       `json:"id"`
	JobID                   string       `json:"job_id"`
	LaborRateID             string       `json:"labor_rate_id"`
	ActivityType            ActivityType `json:"activity_type"`
	Minutes                 int          `json:"minutes"`
	InternalHourlyRateCents int64        `json:"internal_hourly_rate_cents"`
	OccurredAt              time.Time    `json:"occurred_at"`
	RecordedBy              string       `json:"recorded_by"`
	Notes                   string       `json:"notes"`
	CreatedAt               time.Time    `json:"created_at"`
}
type EntryValues struct {
	LaborRateID string
	Minutes     int
	OccurredAt  time.Time
	Notes       string
}
type Summary struct {
	Items             []Entry                `json:"items"`
	TotalMinutes      int64                  `json:"total_minutes"`
	MinutesByActivity map[ActivityType]int64 `json:"minutes_by_activity"`
}
