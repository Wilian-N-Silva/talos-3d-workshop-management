package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
	"github.com/jackc/pgx/v5/pgconn"
)

const materialColumns = `id, manufacturer, name, material_type, color_name, color_hex, nominal_density::text, default_replacement_cost_per_kg_cents, notes, created_at, updated_at`
const spoolColumns = `id, code, material_id, nominal_net_weight_g::text, tare_weight_g::text, gross_weight_at_open_g::text, current_remaining_weight_g::text, purchase_cost_cents, replacement_cost_per_kg_cents, opened_at, last_weighed_at, last_dried_at, storage_location, storage_status, lot_number, status, created_at, updated_at`
const measurementColumns = `id, spool_id, measured_at, gross_weight_g::text, derived_remaining_weight_g::text, source, notes, recorded_by, created_at`

type FilamentInventoryRepository struct{ database *sql.DB }

func NewFilamentInventoryRepository(database *sql.DB) *FilamentInventoryRepository {
	return &FilamentInventoryRepository{database: database}
}

func (repository *FilamentInventoryRepository) CreateMaterial(ctx context.Context, values domain.MaterialValues, now time.Time) (domain.Material, error) {
	result, err := scanMaterial(repository.database.QueryRowContext(ctx, `INSERT INTO materials (manufacturer, name, material_type, color_name, color_hex, nominal_density, default_replacement_cost_per_kg_cents, notes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9) RETURNING `+materialColumns, values.Manufacturer, values.Name, values.MaterialType, values.ColorName, values.ColorHex, values.NominalDensity, values.DefaultReplacementCostPerKgCents, values.Notes, now.UTC()))
	if err != nil {
		return domain.Material{}, fmt.Errorf("insert material: %w", err)
	}
	return result, nil
}
func (repository *FilamentInventoryRepository) FindMaterial(ctx context.Context, id string) (domain.Material, error) {
	result, err := scanMaterial(repository.database.QueryRowContext(ctx, `SELECT `+materialColumns+` FROM materials WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Material{}, domain.ErrMaterialNotFound
	}
	if err != nil {
		return domain.Material{}, fmt.Errorf("find material: %w", err)
	}
	return result, nil
}
func (repository *FilamentInventoryRepository) ListMaterials(ctx context.Context) ([]domain.Material, error) {
	rows, err := repository.database.QueryContext(ctx, `SELECT `+materialColumns+` FROM materials ORDER BY manufacturer,name,id LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("query materials: %w", err)
	}
	defer rows.Close()
	result := []domain.Material{}
	for rows.Next() {
		value, err := scanMaterial(rows)
		if err != nil {
			return nil, fmt.Errorf("scan material: %w", err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate materials: %w", err)
	}
	return result, nil
}
func (repository *FilamentInventoryRepository) UpdateMaterial(ctx context.Context, id string, values domain.MaterialValues, now time.Time) (domain.Material, error) {
	result, err := scanMaterial(repository.database.QueryRowContext(ctx, `UPDATE materials SET manufacturer=$2,name=$3,material_type=$4,color_name=$5,color_hex=$6,nominal_density=$7,default_replacement_cost_per_kg_cents=$8,notes=$9,updated_at=GREATEST(updated_at,$10) WHERE id=$1 RETURNING `+materialColumns, id, values.Manufacturer, values.Name, values.MaterialType, values.ColorName, values.ColorHex, values.NominalDensity, values.DefaultReplacementCostPerKgCents, values.Notes, now.UTC()))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Material{}, domain.ErrMaterialNotFound
	}
	if err != nil {
		return domain.Material{}, fmt.Errorf("update material: %w", err)
	}
	return result, nil
}
func (repository *FilamentInventoryRepository) DeleteMaterial(ctx context.Context, id string) error {
	result, err := repository.database.ExecContext(ctx, "DELETE FROM materials WHERE id=$1", id)
	if fkError(err, "material_spools_material_fk") {
		return domain.ErrInventoryHistoryExists
	}
	if err != nil {
		return fmt.Errorf("delete material: %w", err)
	}
	return requireDeleted(result, domain.ErrMaterialNotFound)
}

func (repository *FilamentInventoryRepository) CreateSpool(ctx context.Context, values domain.SpoolValues, now time.Time) (domain.Spool, error) {
	result, err := scanSpool(repository.database.QueryRowContext(ctx, `INSERT INTO material_spools (code,material_id,nominal_net_weight_g,tare_weight_g,gross_weight_at_open_g,purchase_cost_cents,replacement_cost_per_kg_cents,opened_at,last_dried_at,storage_location,storage_status,lot_number,status,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14) RETURNING `+spoolColumns, values.Code, values.MaterialID, values.NominalNetWeightG, values.TareWeightG, values.GrossWeightAtOpenG, values.PurchaseCostCents, values.ReplacementCostPerKgCents, utcPointer(values.OpenedAt), utcPointer(values.LastDriedAt), values.StorageLocation, values.StorageStatus, values.LotNumber, values.Status, now.UTC()))
	if uniqueError(err, "material_spools_code_unique") {
		return domain.Spool{}, domain.ErrSpoolCodeConflict
	}
	if fkError(err, "material_spools_material_fk") {
		return domain.Spool{}, domain.ErrMaterialNotFound
	}
	if err != nil {
		return domain.Spool{}, fmt.Errorf("insert spool: %w", err)
	}
	return result, nil
}
func (repository *FilamentInventoryRepository) FindSpool(ctx context.Context, id string) (domain.Spool, error) {
	result, err := scanSpool(repository.database.QueryRowContext(ctx, `SELECT `+spoolColumns+` FROM material_spools WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Spool{}, domain.ErrSpoolNotFound
	}
	if err != nil {
		return domain.Spool{}, fmt.Errorf("find spool: %w", err)
	}
	return result, nil
}
func (repository *FilamentInventoryRepository) ListSpools(ctx context.Context) ([]domain.Spool, error) {
	rows, err := repository.database.QueryContext(ctx, `SELECT `+spoolColumns+` FROM material_spools ORDER BY code,id LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("query spools: %w", err)
	}
	defer rows.Close()
	result := []domain.Spool{}
	for rows.Next() {
		value, err := scanSpool(rows)
		if err != nil {
			return nil, fmt.Errorf("scan spool: %w", err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate spools: %w", err)
	}
	return result, nil
}
func (repository *FilamentInventoryRepository) UpdateSpool(ctx context.Context, id string, values domain.SpoolValues, now time.Time) (domain.Spool, error) {
	result, err := scanSpool(repository.database.QueryRowContext(ctx, `UPDATE material_spools SET code=$2,material_id=$3,nominal_net_weight_g=$4,tare_weight_g=$5,gross_weight_at_open_g=$6,purchase_cost_cents=$7,replacement_cost_per_kg_cents=$8,opened_at=$9,last_dried_at=$10,storage_location=$11,storage_status=$12,lot_number=$13,status=$14,updated_at=GREATEST(updated_at,$15) WHERE id=$1 RETURNING `+spoolColumns, id, values.Code, values.MaterialID, values.NominalNetWeightG, values.TareWeightG, values.GrossWeightAtOpenG, values.PurchaseCostCents, values.ReplacementCostPerKgCents, utcPointer(values.OpenedAt), utcPointer(values.LastDriedAt), values.StorageLocation, values.StorageStatus, values.LotNumber, values.Status, now.UTC()))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Spool{}, domain.ErrSpoolNotFound
	}
	if uniqueError(err, "material_spools_code_unique") {
		return domain.Spool{}, domain.ErrSpoolCodeConflict
	}
	if fkError(err, "material_spools_material_fk") {
		return domain.Spool{}, domain.ErrMaterialNotFound
	}
	if err != nil {
		return domain.Spool{}, fmt.Errorf("update spool: %w", err)
	}
	return result, nil
}
func (repository *FilamentInventoryRepository) DeleteSpool(ctx context.Context, id string) error {
	result, err := repository.database.ExecContext(ctx, "DELETE FROM material_spools WHERE id=$1", id)
	if fkError(err, "spool_measurements_spool_fk") {
		return domain.ErrInventoryHistoryExists
	}
	if err != nil {
		return fmt.Errorf("delete spool: %w", err)
	}
	return requireDeleted(result, domain.ErrSpoolNotFound)
}

func (repository *FilamentInventoryRepository) RecordMeasurement(ctx context.Context, spoolID, actorID string, values domain.MeasurementValues, createdAt time.Time) (domain.SpoolMeasurement, error) {
	tx, err := repository.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return domain.SpoolMeasurement{}, fmt.Errorf("begin measurement transaction: %w", err)
	}
	defer tx.Rollback()
	var tare string
	if err := tx.QueryRowContext(ctx, "SELECT tare_weight_g::text FROM material_spools WHERE id=$1 FOR UPDATE", spoolID).Scan(&tare); errors.Is(err, sql.ErrNoRows) {
		return domain.SpoolMeasurement{}, domain.ErrSpoolNotFound
	} else if err != nil {
		return domain.SpoolMeasurement{}, fmt.Errorf("lock spool: %w", err)
	}
	gross, _ := new(big.Rat).SetString(values.GrossWeightG)
	tareValue, _ := new(big.Rat).SetString(tare)
	if gross.Cmp(tareValue) < 0 {
		return domain.SpoolMeasurement{}, domain.ErrMeasurementBelowTare
	}
	measurement, err := scanMeasurement(tx.QueryRowContext(ctx, `INSERT INTO spool_measurements (spool_id,measured_at,gross_weight_g,derived_remaining_weight_g,source,notes,recorded_by,created_at) VALUES ($1,$2,$3,CAST($3 AS NUMERIC)-CAST($4 AS NUMERIC),$5,$6,$7,$8) RETURNING `+measurementColumns, spoolID, values.MeasuredAt.UTC(), values.GrossWeightG, tare, values.Source, values.Notes, actorID, createdAt.UTC()))
	if err != nil {
		return domain.SpoolMeasurement{}, fmt.Errorf("insert measurement: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE material_spools
		SET current_remaining_weight_g=$2,last_weighed_at=$3,updated_at=GREATEST(updated_at,$4)
		WHERE id=$1 AND (last_weighed_at IS NULL OR last_weighed_at <= $3)`, spoolID, measurement.DerivedRemainingWeightG, values.MeasuredAt.UTC(), createdAt.UTC()); err != nil {
		return domain.SpoolMeasurement{}, fmt.Errorf("update spool measurement cache: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.SpoolMeasurement{}, fmt.Errorf("commit measurement: %w", err)
	}
	return measurement, nil
}
func (repository *FilamentInventoryRepository) ListMeasurements(ctx context.Context, spoolID string) ([]domain.SpoolMeasurement, error) {
	var exists bool
	if err := repository.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM material_spools WHERE id=$1)", spoolID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check spool: %w", err)
	}
	if !exists {
		return nil, domain.ErrSpoolNotFound
	}
	rows, err := repository.database.QueryContext(ctx, `SELECT `+measurementColumns+` FROM spool_measurements WHERE spool_id=$1 ORDER BY measured_at DESC,id DESC LIMIT 100`, spoolID)
	if err != nil {
		return nil, fmt.Errorf("query measurements: %w", err)
	}
	defer rows.Close()
	result := []domain.SpoolMeasurement{}
	for rows.Next() {
		value, err := scanMeasurement(rows)
		if err != nil {
			return nil, fmt.Errorf("scan measurement: %w", err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate measurements: %w", err)
	}
	return result, nil
}

func scanMaterial(row rowScanner) (domain.Material, error) {
	var value domain.Material
	var color sql.NullString
	if err := row.Scan(&value.ID, &value.Manufacturer, &value.Name, &value.MaterialType, &value.ColorName, &color, &value.NominalDensity, &value.DefaultReplacementCostPerKgCents, &value.Notes, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return domain.Material{}, err
	}
	if color.Valid {
		value.ColorHex = &color.String
	}
	value.CreatedAt, value.UpdatedAt = value.CreatedAt.UTC(), value.UpdatedAt.UTC()
	return value, nil
}
func scanSpool(row rowScanner) (domain.Spool, error) {
	var value domain.Spool
	var gross, current sql.NullString
	var opened, weighed, dried sql.NullTime
	if err := row.Scan(&value.ID, &value.Code, &value.MaterialID, &value.NominalNetWeightG, &value.TareWeightG, &gross, &current, &value.PurchaseCostCents, &value.ReplacementCostPerKgCents, &opened, &weighed, &dried, &value.StorageLocation, &value.StorageStatus, &value.LotNumber, &value.Status, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return domain.Spool{}, err
	}
	if gross.Valid {
		value.GrossWeightAtOpenG = &gross.String
	}
	if current.Valid {
		value.CurrentRemainingWeightG = &current.String
	}
	value.OpenedAt = timePointer(opened)
	value.LastWeighedAt = timePointer(weighed)
	value.LastDriedAt = timePointer(dried)
	value.CreatedAt, value.UpdatedAt = value.CreatedAt.UTC(), value.UpdatedAt.UTC()
	return value, nil
}
func scanMeasurement(row rowScanner) (domain.SpoolMeasurement, error) {
	var value domain.SpoolMeasurement
	if err := row.Scan(&value.ID, &value.SpoolID, &value.MeasuredAt, &value.GrossWeightG, &value.DerivedRemainingWeightG, &value.Source, &value.Notes, &value.RecordedBy, &value.CreatedAt); err != nil {
		return domain.SpoolMeasurement{}, err
	}
	value.MeasuredAt, value.CreatedAt = value.MeasuredAt.UTC(), value.CreatedAt.UTC()
	return value, nil
}
func requireDeleted(result sql.Result, notFound error) error {
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted row count: %w", err)
	}
	if count == 0 {
		return notFound
	}
	return nil
}
func uniqueError(err error, constraint string) bool {
	var value *pgconn.PgError
	return errors.As(err, &value) && value.Code == "23505" && value.ConstraintName == constraint
}
func fkError(err error, constraint string) bool {
	var value *pgconn.PgError
	return errors.As(err, &value) && (value.Code == "23503" || value.Code == "23001") && value.ConstraintName == constraint
}
func utcPointer(value *time.Time) any {
	if value == nil {
		return nil
	}
	result := value.UTC()
	return result
}
func timePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time.UTC()
	return &result
}
