package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/printers"
)

const printerColumns = `id,name,manufacturer,model,nozzle_diameter::text,location,acquisition_cost_cents,residual_value_cents,useful_life_hours::text,maintenance_reserve_per_hour_cents,status,notes,created_at,updated_at`

type PrinterRepository struct {
	database *sql.DB
}

func NewPrinterRepository(database *sql.DB) *PrinterRepository {
	return &PrinterRepository{database: database}
}

func (repository *PrinterRepository) Create(ctx context.Context, values domain.Values, now time.Time) (domain.Printer, error) {
	printer, err := scanPrinter(repository.database.QueryRowContext(ctx, `
		INSERT INTO printers (name,manufacturer,model,nozzle_diameter,location,acquisition_cost_cents,residual_value_cents,useful_life_hours,maintenance_reserve_per_hour_cents,status,notes,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12)
		RETURNING `+printerColumns,
		values.Name, values.Manufacturer, values.Model, values.NozzleDiameter, values.Location,
		values.AcquisitionCostCents, values.ResidualValueCents, values.UsefulLifeHours,
		values.MaintenanceReservePerHourCents, values.Status, values.Notes, now.UTC(),
	))
	if uniqueError(err, "printers_name_unique") {
		return domain.Printer{}, domain.ErrPrinterNameConflict
	}
	if err != nil {
		return domain.Printer{}, fmt.Errorf("insert printer: %w", err)
	}
	return printer, nil
}

func (repository *PrinterRepository) FindByID(ctx context.Context, id string) (domain.Printer, error) {
	printer, err := scanPrinter(repository.database.QueryRowContext(ctx, `SELECT `+printerColumns+` FROM printers WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Printer{}, domain.ErrPrinterNotFound
	}
	if err != nil {
		return domain.Printer{}, fmt.Errorf("find printer: %w", err)
	}
	return printer, nil
}

func (repository *PrinterRepository) List(ctx context.Context) ([]domain.Printer, error) {
	rows, err := repository.database.QueryContext(ctx, `SELECT `+printerColumns+` FROM printers ORDER BY name,id LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("query printers: %w", err)
	}
	defer rows.Close()
	printers := []domain.Printer{}
	for rows.Next() {
		printer, err := scanPrinter(rows)
		if err != nil {
			return nil, fmt.Errorf("scan printer: %w", err)
		}
		printers = append(printers, printer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate printers: %w", err)
	}
	return printers, nil
}

func (repository *PrinterRepository) Update(ctx context.Context, id string, values domain.Values, now time.Time) (domain.Printer, error) {
	printer, err := scanPrinter(repository.database.QueryRowContext(ctx, `
		UPDATE printers SET name=$2,manufacturer=$3,model=$4,nozzle_diameter=$5,location=$6,
		acquisition_cost_cents=$7,residual_value_cents=$8,useful_life_hours=$9,
		maintenance_reserve_per_hour_cents=$10,status=$11,notes=$12,updated_at=GREATEST(updated_at,$13)
		WHERE id=$1 RETURNING `+printerColumns,
		id, values.Name, values.Manufacturer, values.Model, values.NozzleDiameter, values.Location,
		values.AcquisitionCostCents, values.ResidualValueCents, values.UsefulLifeHours,
		values.MaintenanceReservePerHourCents, values.Status, values.Notes, now.UTC(),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Printer{}, domain.ErrPrinterNotFound
	}
	if uniqueError(err, "printers_name_unique") {
		return domain.Printer{}, domain.ErrPrinterNameConflict
	}
	if err != nil {
		return domain.Printer{}, fmt.Errorf("update printer: %w", err)
	}
	return printer, nil
}

func (repository *PrinterRepository) Delete(ctx context.Context, id string) error {
	result, err := repository.database.ExecContext(ctx, "DELETE FROM printers WHERE id=$1", id)
	if err != nil {
		return fmt.Errorf("delete printer: %w", err)
	}
	return requireDeleted(result, domain.ErrPrinterNotFound)
}

func scanPrinter(row rowScanner) (domain.Printer, error) {
	var printer domain.Printer
	if err := row.Scan(
		&printer.ID, &printer.Name, &printer.Manufacturer, &printer.Model, &printer.NozzleDiameter,
		&printer.Location, &printer.AcquisitionCostCents, &printer.ResidualValueCents,
		&printer.UsefulLifeHours, &printer.MaintenanceReservePerHourCents, &printer.Status,
		&printer.Notes, &printer.CreatedAt, &printer.UpdatedAt,
	); err != nil {
		return domain.Printer{}, err
	}
	printer.CreatedAt = printer.CreatedAt.UTC()
	printer.UpdatedAt = printer.UpdatedAt.UTC()
	return printer, nil
}
