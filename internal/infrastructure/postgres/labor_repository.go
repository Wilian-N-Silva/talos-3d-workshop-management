package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domainjobs "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/labor"
)

const laborRateColumns = `id,name,activity_type,cost_hourly_rate_cents,active,created_at,updated_at`
const laborEntryColumns = `id,job_id,labor_rate_id,activity_type,minutes,internal_hourly_rate_cents,occurred_at,recorded_by,notes,created_at`

type LaborRepository struct{ database *sql.DB }

func NewLaborRepository(database *sql.DB) *LaborRepository {
	return &LaborRepository{database: database}
}

func (repository *LaborRepository) CreateRate(ctx context.Context, values domain.RateValues, now time.Time) (domain.Rate, error) {
	rate, err := scanLaborRate(repository.database.QueryRowContext(ctx, `INSERT INTO labor_rates(name,activity_type,cost_hourly_rate_cents,active,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$5) RETURNING `+laborRateColumns, values.Name, values.ActivityType, values.CostHourlyRateCents, values.Active, now.UTC()))
	if uniqueError(err, "labor_rates_name_unique") {
		return domain.Rate{}, domain.ErrRateConflict
	}
	if err != nil {
		return domain.Rate{}, fmt.Errorf("insert labor rate: %w", err)
	}
	return rate, nil
}
func (repository *LaborRepository) ListRates(ctx context.Context) ([]domain.Rate, error) {
	rows, err := repository.database.QueryContext(ctx, `SELECT `+laborRateColumns+` FROM labor_rates ORDER BY active DESC,activity_type,name,id`)
	if err != nil {
		return nil, fmt.Errorf("query labor rates: %w", err)
	}
	defer rows.Close()
	rates := []domain.Rate{}
	for rows.Next() {
		rate, err := scanLaborRate(rows)
		if err != nil {
			return nil, fmt.Errorf("scan labor rate: %w", err)
		}
		rates = append(rates, rate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate labor rates: %w", err)
	}
	return rates, nil
}
func (repository *LaborRepository) UpdateRate(ctx context.Context, id string, values domain.RateValues, now time.Time) (domain.Rate, error) {
	rate, err := scanLaborRate(repository.database.QueryRowContext(ctx, `UPDATE labor_rates SET name=$2,activity_type=$3,cost_hourly_rate_cents=$4,active=$5,updated_at=GREATEST(updated_at,$6) WHERE id=$1 RETURNING `+laborRateColumns, id, values.Name, values.ActivityType, values.CostHourlyRateCents, values.Active, now.UTC()))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Rate{}, domain.ErrRateNotFound
	}
	if uniqueError(err, "labor_rates_name_unique") {
		return domain.Rate{}, domain.ErrRateConflict
	}
	if err != nil {
		return domain.Rate{}, fmt.Errorf("update labor rate: %w", err)
	}
	return rate, nil
}
func (repository *LaborRepository) CreateEntry(ctx context.Context, jobID, recordedBy string, values domain.EntryValues, createdAt time.Time) (domain.Entry, error) {
	entry, err := scanLaborEntry(repository.database.QueryRowContext(ctx, `INSERT INTO job_labor_entries(job_id,labor_rate_id,activity_type,minutes,internal_hourly_rate_cents,occurred_at,recorded_by,notes,created_at) SELECT $1,r.id,r.activity_type,$3,r.cost_hourly_rate_cents,$4,$5,$6,$7 FROM labor_rates r WHERE r.id=$2 AND r.active AND EXISTS(SELECT 1 FROM print_jobs WHERE id=$1) RETURNING `+laborEntryColumns, jobID, values.LaborRateID, values.Minutes, values.OccurredAt.UTC(), recordedBy, values.Notes, createdAt.UTC()))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Entry{}, repository.entryReferenceError(ctx, jobID, values.LaborRateID)
	}
	if foreignKeyViolation(err, "job_labor_entries_recorded_by_fk") {
		return domain.Entry{}, domainjobs.ErrJobReference
	}
	if err != nil {
		return domain.Entry{}, fmt.Errorf("insert labor entry: %w", err)
	}
	return entry, nil
}
func (repository *LaborRepository) ListEntries(ctx context.Context, jobID string) ([]domain.Entry, error) {
	var exists bool
	if err := repository.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM print_jobs WHERE id=$1)", jobID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check print job: %w", err)
	}
	if !exists {
		return nil, domainjobs.ErrJobNotFound
	}
	rows, err := repository.database.QueryContext(ctx, `SELECT `+laborEntryColumns+` FROM job_labor_entries WHERE job_id=$1 ORDER BY occurred_at DESC,id DESC`, jobID)
	if err != nil {
		return nil, fmt.Errorf("query labor entries: %w", err)
	}
	defer rows.Close()
	entries := []domain.Entry{}
	for rows.Next() {
		entry, err := scanLaborEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scan labor entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate labor entries: %w", err)
	}
	return entries, nil
}
func (repository *LaborRepository) entryReferenceError(ctx context.Context, jobID, rateID string) error {
	var jobExists bool
	if err := repository.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM print_jobs WHERE id=$1)", jobID).Scan(&jobExists); err != nil {
		return err
	}
	if !jobExists {
		return domainjobs.ErrJobNotFound
	}
	return domain.ErrRateNotFound
}
func scanLaborRate(row rowScanner) (domain.Rate, error) {
	var value domain.Rate
	err := row.Scan(&value.ID, &value.Name, &value.ActivityType, &value.CostHourlyRateCents, &value.Active, &value.CreatedAt, &value.UpdatedAt)
	value.CreatedAt = value.CreatedAt.UTC()
	value.UpdatedAt = value.UpdatedAt.UTC()
	return value, err
}
func scanLaborEntry(row rowScanner) (domain.Entry, error) {
	var value domain.Entry
	err := row.Scan(&value.ID, &value.JobID, &value.LaborRateID, &value.ActivityType, &value.Minutes, &value.InternalHourlyRateCents, &value.OccurredAt, &value.RecordedBy, &value.Notes, &value.CreatedAt)
	value.OccurredAt = value.OccurredAt.UTC()
	value.CreatedAt = value.CreatedAt.UTC()
	return value, err
}
