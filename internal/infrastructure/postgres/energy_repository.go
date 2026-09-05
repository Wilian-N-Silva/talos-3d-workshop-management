package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/energy"
	domainjobs "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
)

const energyMeasurementColumns = `id,job_id,source,meter_start_kwh::text,meter_end_kwh::text,measured_kwh::text,estimated_average_power_w::text,energy_rate_cents_per_kwh,occurred_at,recorded_by,notes`

type EnergyRepository struct{ database *sql.DB }

func NewEnergyRepository(database *sql.DB) *EnergyRepository {
	return &EnergyRepository{database: database}
}

func (repository *EnergyRepository) Create(ctx context.Context, jobID, recordedBy string, values domain.Values) (domain.Measurement, error) {
	row := repository.database.QueryRowContext(ctx, `INSERT INTO energy_measurements(job_id,source,meter_start_kwh,meter_end_kwh,measured_kwh,estimated_average_power_w,energy_rate_cents_per_kwh,occurred_at,recorded_by,notes) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10 WHERE EXISTS(SELECT 1 FROM print_jobs WHERE id=$1) RETURNING `+energyMeasurementColumns, jobID, values.Source, values.MeterStartKWh, values.MeterEndKWh, values.MeasuredKWh, values.EstimatedAveragePowerW, values.EnergyRateCentsPerKWh, values.OccurredAt.UTC(), recordedBy, values.Notes)
	measurement, err := scanEnergyMeasurement(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Measurement{}, domainjobs.ErrJobNotFound
	}
	if foreignKeyViolation(err, "energy_measurements_recorded_by_fk") {
		return domain.Measurement{}, domainjobs.ErrJobReference
	}
	if err != nil {
		return domain.Measurement{}, fmt.Errorf("insert energy measurement: %w", err)
	}
	return measurement, nil
}

func (repository *EnergyRepository) List(ctx context.Context, jobID string) ([]domain.Measurement, error) {
	var exists bool
	if err := repository.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM print_jobs WHERE id=$1)", jobID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check print job: %w", err)
	}
	if !exists {
		return nil, domainjobs.ErrJobNotFound
	}
	rows, err := repository.database.QueryContext(ctx, `SELECT `+energyMeasurementColumns+` FROM energy_measurements WHERE job_id=$1 ORDER BY occurred_at DESC,id DESC`, jobID)
	if err != nil {
		return nil, fmt.Errorf("query energy measurements: %w", err)
	}
	defer rows.Close()
	measurements := []domain.Measurement{}
	for rows.Next() {
		measurement, err := scanEnergyMeasurement(rows)
		if err != nil {
			return nil, fmt.Errorf("scan energy measurement: %w", err)
		}
		measurements = append(measurements, measurement)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate energy measurements: %w", err)
	}
	return measurements, nil
}

func scanEnergyMeasurement(row rowScanner) (domain.Measurement, error) {
	var measurement domain.Measurement
	var start, end, measured, power sql.NullString
	if err := row.Scan(&measurement.ID, &measurement.JobID, &measurement.Source, &start, &end, &measured, &power, &measurement.EnergyRateCentsPerKWh, &measurement.OccurredAt, &measurement.RecordedBy, &measurement.Notes); err != nil {
		return domain.Measurement{}, err
	}
	measurement.MeterStartKWh = canonicalEnergyDecimal(start)
	measurement.MeterEndKWh = canonicalEnergyDecimal(end)
	measurement.MeasuredKWh = canonicalEnergyDecimal(measured)
	measurement.EstimatedAveragePowerW = canonicalEnergyDecimal(power)
	measurement.OccurredAt = measurement.OccurredAt.UTC()
	return measurement, nil
}

func canonicalEnergyDecimal(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	canonical := value.String
	if strings.Contains(canonical, ".") {
		canonical = strings.TrimRight(strings.TrimRight(canonical, "0"), ".")
	}
	if canonical == "" {
		canonical = "0"
	}
	return &canonical
}
