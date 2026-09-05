package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/maintenance"
	domainprinters "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/printers"
	"strings"
	"time"
)

const maintenanceColumns = `id,printer_id,type,performed_at,printer_hours::text,description,cost_cents,downtime_minutes,notes,created_by,created_at`

type MaintenanceRepository struct{ database *sql.DB }

func NewMaintenanceRepository(database *sql.DB) *MaintenanceRepository {
	return &MaintenanceRepository{database: database}
}
func (repository *MaintenanceRepository) Create(ctx context.Context, printerID, createdBy string, values domain.Values, createdAt time.Time) (domain.Event, error) {
	event, err := scanMaintenanceEvent(repository.database.QueryRowContext(ctx, `INSERT INTO maintenance_events(printer_id,type,performed_at,printer_hours,description,cost_cents,downtime_minutes,notes,created_by,created_at) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10 WHERE EXISTS(SELECT 1 FROM printers WHERE id=$1) RETURNING `+maintenanceColumns, printerID, values.Type, values.PerformedAt.UTC(), values.PrinterHours, values.Description, values.CostCents, values.DowntimeMinutes, values.Notes, createdBy, createdAt.UTC()))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Event{}, domainprinters.ErrPrinterNotFound
	}
	if foreignKeyViolation(err, "maintenance_events_created_by_fk") {
		return domain.Event{}, domain.ErrMaintenanceReference
	}
	if err != nil {
		return domain.Event{}, fmt.Errorf("insert maintenance event: %w", err)
	}
	return event, nil
}
func (repository *MaintenanceRepository) List(ctx context.Context, printerID string) ([]domain.Event, error) {
	var exists bool
	if err := repository.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM printers WHERE id=$1)", printerID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check printer: %w", err)
	}
	if !exists {
		return nil, domainprinters.ErrPrinterNotFound
	}
	rows, err := repository.database.QueryContext(ctx, `SELECT `+maintenanceColumns+` FROM maintenance_events WHERE printer_id=$1 ORDER BY performed_at DESC,id DESC`, printerID)
	if err != nil {
		return nil, fmt.Errorf("query maintenance events: %w", err)
	}
	defer rows.Close()
	events := []domain.Event{}
	for rows.Next() {
		event, err := scanMaintenanceEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan maintenance event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate maintenance events: %w", err)
	}
	return events, nil
}
func scanMaintenanceEvent(row rowScanner) (domain.Event, error) {
	var event domain.Event
	var hours sql.NullString
	var cost sql.NullInt64
	err := row.Scan(&event.ID, &event.PrinterID, &event.Type, &event.PerformedAt, &hours, &event.Description, &cost, &event.DowntimeMinutes, &event.Notes, &event.CreatedBy, &event.CreatedAt)
	if hours.Valid {
		value := hours.String
		if strings.Contains(value, ".") {
			value = strings.TrimRight(strings.TrimRight(value, "0"), ".")
		}
		event.PrinterHours = &value
	}
	if cost.Valid {
		event.CostCents = &cost.Int64
	}
	event.PerformedAt = event.PerformedAt.UTC()
	event.CreatedAt = event.CreatedAt.UTC()
	return event, err
}
