package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	domaininventory "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
)

const materialUsageColumns = `id,print_job_id,material_id,spool_id,role,planned_grams::text,actual_grams::text,planned_meters::text,actual_meters::text,measurement_source,historical_material_cost_cents,replacement_material_cost_cents,created_at,updated_at`

func (r *JobRepository) CreateMaterialUsage(ctx context.Context, jobID string, v domain.MaterialUsageValues, now time.Time) (domain.MaterialUsage, error) {
	row := r.database.QueryRowContext(ctx, `INSERT INTO print_job_material_usage(print_job_id,material_id,spool_id,role,planned_grams,actual_grams,planned_meters,actual_meters,measurement_source,historical_material_cost_cents,replacement_material_cost_cents,created_at,updated_at) SELECT $1,s.material_id,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11 FROM material_spools s WHERE s.id=$2 AND EXISTS(SELECT 1 FROM print_jobs j WHERE j.id=$1 AND j.status IN ('draft','prepared','printing','awaiting_review')) RETURNING `+materialUsageColumns, jobID, v.SpoolID, v.Role, v.PlannedGrams, v.ActualGrams, v.PlannedMeters, v.ActualMeters, v.MeasurementSource, v.HistoricalMaterialCostCents, v.ReplacementMaterialCostCents, now.UTC())
	result, err := scanMaterialUsage(row)
	if uniqueError(err, "job_material_role_spool_unique") {
		return domain.MaterialUsage{}, domain.ErrMaterialUsageConflict
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.MaterialUsage{}, r.materialUsageReferenceError(ctx, jobID, v.SpoolID)
	}
	if err != nil {
		return domain.MaterialUsage{}, fmt.Errorf("insert job material usage: %w", err)
	}
	return result, nil
}
func (r *JobRepository) ListMaterialUsage(ctx context.Context, jobID string) ([]domain.MaterialUsage, error) {
	var exists bool
	if err := r.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM print_jobs WHERE id=$1)", jobID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check print job: %w", err)
	}
	if !exists {
		return nil, domain.ErrJobNotFound
	}
	rows, err := r.database.QueryContext(ctx, `SELECT `+materialUsageColumns+` FROM print_job_material_usage WHERE print_job_id=$1 ORDER BY role,id`, jobID)
	if err != nil {
		return nil, fmt.Errorf("query job material usage: %w", err)
	}
	defer rows.Close()
	result := []domain.MaterialUsage{}
	for rows.Next() {
		value, err := scanMaterialUsage(rows)
		if err != nil {
			return nil, fmt.Errorf("scan job material usage: %w", err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate job material usage: %w", err)
	}
	return result, nil
}
func (r *JobRepository) UpdateMaterialUsage(ctx context.Context, jobID, id string, v domain.MaterialUsageValues, now time.Time) (domain.MaterialUsage, error) {
	row := r.database.QueryRowContext(ctx, `UPDATE print_job_material_usage u SET material_id=s.material_id,spool_id=$3,role=$4,planned_grams=$5,actual_grams=$6,planned_meters=$7,actual_meters=$8,measurement_source=$9,updated_at=GREATEST(u.updated_at,$10) FROM material_spools s WHERE u.id=$2 AND u.print_job_id=$1 AND s.id=$3 AND EXISTS(SELECT 1 FROM print_jobs j WHERE j.id=$1 AND j.status IN ('draft','prepared','printing','awaiting_review')) RETURNING `+qualifiedMaterialUsageColumns("u"), jobID, id, v.SpoolID, v.Role, v.PlannedGrams, v.ActualGrams, v.PlannedMeters, v.ActualMeters, v.MeasurementSource, now.UTC())
	result, err := scanMaterialUsage(row)
	if uniqueError(err, "job_material_role_spool_unique") {
		return domain.MaterialUsage{}, domain.ErrMaterialUsageConflict
	}
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if err := r.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM print_job_material_usage WHERE id=$1 AND print_job_id=$2)", id, jobID).Scan(&exists); err != nil {
			return domain.MaterialUsage{}, fmt.Errorf("check job material usage: %w", err)
		}
		if !exists {
			return domain.MaterialUsage{}, domain.ErrMaterialUsageNotFound
		}
		return domain.MaterialUsage{}, r.materialUsageReferenceError(ctx, jobID, v.SpoolID)
	}
	if err != nil {
		return domain.MaterialUsage{}, fmt.Errorf("update job material usage: %w", err)
	}
	return result, nil
}
func (r *JobRepository) DeleteMaterialUsage(ctx context.Context, jobID, id string) error {
	result, err := r.database.ExecContext(ctx, `DELETE FROM print_job_material_usage u USING print_jobs j WHERE u.id=$2 AND u.print_job_id=$1 AND j.id=$1 AND j.status IN ('draft','prepared')`, jobID, id)
	if err != nil {
		return fmt.Errorf("delete job material usage: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted usage count: %w", err)
	}
	if rows == 0 {
		var exists bool
		if err := r.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM print_job_material_usage WHERE id=$1 AND print_job_id=$2)", id, jobID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return domain.ErrMaterialUsageNotFound
		}
		return domain.ErrJobNotEditable
	}
	return nil
}
func (r *JobRepository) materialUsageReferenceError(ctx context.Context, jobID, spoolID string) error {
	var jobExists, editable, spoolExists bool
	if err := r.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM print_jobs WHERE id=$1),EXISTS(SELECT 1 FROM print_jobs WHERE id=$1 AND status IN ('draft','prepared','printing','awaiting_review')),EXISTS(SELECT 1 FROM material_spools WHERE id=$2)", jobID, spoolID).Scan(&jobExists, &editable, &spoolExists); err != nil {
		return fmt.Errorf("check material usage references: %w", err)
	}
	if !jobExists {
		return domain.ErrJobNotFound
	}
	if !editable {
		return domain.ErrJobNotEditable
	}
	if !spoolExists {
		return domaininventory.ErrSpoolNotFound
	}
	return domain.ErrMaterialUsageNotFound
}
func qualifiedMaterialUsageColumns(alias string) string {
	return alias + `.id,` + alias + `.print_job_id,` + alias + `.material_id,` + alias + `.spool_id,` + alias + `.role,` + alias + `.planned_grams::text,` + alias + `.actual_grams::text,` + alias + `.planned_meters::text,` + alias + `.actual_meters::text,` + alias + `.measurement_source,` + alias + `.historical_material_cost_cents,` + alias + `.replacement_material_cost_cents,` + alias + `.created_at,` + alias + `.updated_at`
}
func scanMaterialUsage(row rowScanner) (domain.MaterialUsage, error) {
	var value domain.MaterialUsage
	var actual, plannedMeters, actualMeters sql.NullString
	var historical, replacement sql.NullInt64
	if err := row.Scan(&value.ID, &value.PrintJobID, &value.MaterialID, &value.SpoolID, &value.Role, &value.PlannedGrams, &actual, &plannedMeters, &actualMeters, &value.MeasurementSource, &historical, &replacement, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return domain.MaterialUsage{}, err
	}
	if actual.Valid {
		actual.String = canonicalMaterialUsageDecimal(actual.String)
		value.ActualGrams = &actual.String
	}
	if plannedMeters.Valid {
		plannedMeters.String = canonicalMaterialUsageDecimal(plannedMeters.String)
		value.PlannedMeters = &plannedMeters.String
	}
	if actualMeters.Valid {
		actualMeters.String = canonicalMaterialUsageDecimal(actualMeters.String)
		value.ActualMeters = &actualMeters.String
	}
	if historical.Valid {
		value.HistoricalMaterialCostCents = &historical.Int64
	}
	if replacement.Valid {
		value.ReplacementMaterialCostCents = &replacement.Int64
	}
	value.CreatedAt = value.CreatedAt.UTC()
	value.UpdatedAt = value.UpdatedAt.UTC()
	value.PlannedGrams = canonicalMaterialUsageDecimal(value.PlannedGrams)
	return value, nil
}

func canonicalMaterialUsageDecimal(value string) string {
	if strings.Contains(value, ".") {
		value = strings.TrimRight(strings.TrimRight(value, "0"), ".")
	}
	if value == "" {
		return "0"
	}
	return value
}
