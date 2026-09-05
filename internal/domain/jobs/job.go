package jobs

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrJobNotFound      = errors.New("print job not found")
	ErrJobCodeConflict  = errors.New("print job code already exists")
	ErrJobReference     = errors.New("invalid print job reference")
	ErrJobStateConflict = errors.New("print job state changed")
	ErrJobNotEditable   = errors.New("print job is not editable")
	ErrJobNotDeletable  = errors.New("only draft print jobs can be deleted")
)

type Purpose string

const (
	PurposeTest        Purpose = "test"
	PurposePrototype   Purpose = "prototype"
	PurposeProduction  Purpose = "production"
	PurposeMaintenance Purpose = "maintenance"
	PurposeInternal    Purpose = "internal"
	PurposePersonal    Purpose = "personal"
)

type Status string

const (
	StatusDraft          Status = "draft"
	StatusPrepared       Status = "prepared"
	StatusPrinting       Status = "printing"
	StatusAwaitingReview Status = "awaiting_review"
	StatusCompleted      Status = "completed"
	StatusFailed         Status = "failed"
	StatusCancelled      Status = "cancelled"
)

type QualityStatus string

const (
	QualityPending  QualityStatus = "pending"
	QualityApproved QualityStatus = "approved"
	QualityPartial  QualityStatus = "partial"
	QualityFailed   QualityStatus = "failed"
)

type EventType string

const (
	EventCreated               EventType = "created"
	EventPrepared              EventType = "prepared"
	EventPrintingStartedManual EventType = "printing_started_manual"
	EventFinishedManual        EventType = "finished_manual"
	EventReviewed              EventType = "reviewed"
	EventFailed                EventType = "failed"
	EventCancelled             EventType = "cancelled"
)

type Job struct {
	ID              string        `json:"id"`
	Code            string        `json:"code"`
	CatalogItemID   string        `json:"catalog_item_id"`
	DesignVersionID string        `json:"design_version_id"`
	PrinterID       string        `json:"printer_id"`
	OrderItemID     *string       `json:"order_item_id"`
	Purpose         Purpose       `json:"purpose"`
	Status          Status        `json:"status"`
	PlannedQuantity int           `json:"planned_quantity"`
	GoodQuantity    int           `json:"good_quantity"`
	ScrapQuantity   int           `json:"scrap_quantity"`
	Hypothesis      string        `json:"hypothesis"`
	ResultNotes     string        `json:"result_notes"`
	QualityStatus   QualityStatus `json:"quality_status"`
	PlannedSeconds  int64         `json:"planned_seconds"`
	ActualSeconds   *int64        `json:"actual_seconds"`
	LaborMinutes    int           `json:"labor_minutes"`
	CreatedBy       string        `json:"created_by"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	StartedAt       *time.Time    `json:"started_at"`
	CompletedAt     *time.Time    `json:"completed_at"`
}

type Values struct {
	Code            string
	CatalogItemID   string
	DesignVersionID string
	PrinterID       string
	Purpose         Purpose
	PlannedQuantity int
	Hypothesis      string
	PlannedSeconds  int64
	LaborMinutes    int
}

type TransitionValues struct {
	Status        Status
	ActualSeconds *int64
	ResultNotes   string
}

type ReviewValues struct {
	QualityStatus QualityStatus
	GoodQuantity  int
	ScrapQuantity int
	ResultNotes   string
}

type Actor struct{ UserID, DeviceID string }

type Event struct {
	ID             string          `json:"id"`
	JobID          string          `json:"job_id"`
	Type           EventType       `json:"event_type"`
	OccurredAt     time.Time       `json:"occurred_at"`
	ActorUserID    string          `json:"actor_user_id"`
	SourceDeviceID string          `json:"source_device_id"`
	Metadata       json.RawMessage `json:"metadata"`
}
