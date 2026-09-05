package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
)

const supplyColumns = `id, name, sku, unit, current_quantity::text, replacement_unit_cost_cents, minimum_quantity::text, notes, created_at, updated_at`
const supplyMovementColumns = `id, supply_id, type, quantity::text, unit_cost_cents, reference_type, reference_id, occurred_at, recorded_by, notes, created_at`

type SupplyInventoryRepository struct {
	database *sql.DB
}

func NewSupplyInventoryRepository(database *sql.DB) *SupplyInventoryRepository {
	return &SupplyInventoryRepository{database: database}
}

func (repository *SupplyInventoryRepository) CreateSupply(ctx context.Context, values domain.SupplyValues, now time.Time) (domain.Supply, error) {
	result, err := scanSupply(repository.database.QueryRowContext(ctx, `
		INSERT INTO supplies (name, sku, unit, replacement_unit_cost_cents, minimum_quantity, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$7)
		RETURNING `+supplyColumns,
		values.Name, values.SKU, values.Unit, values.ReplacementUnitCostCents, values.MinimumQuantity, values.Notes, now.UTC(),
	))
	if uniqueError(err, "supplies_sku_unique") {
		return domain.Supply{}, domain.ErrSupplySKUConflict
	}
	if err != nil {
		return domain.Supply{}, fmt.Errorf("insert supply: %w", err)
	}
	return result, nil
}

func (repository *SupplyInventoryRepository) FindSupply(ctx context.Context, id string) (domain.Supply, error) {
	result, err := scanSupply(repository.database.QueryRowContext(ctx, `SELECT `+supplyColumns+` FROM supplies WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Supply{}, domain.ErrSupplyNotFound
	}
	if err != nil {
		return domain.Supply{}, fmt.Errorf("find supply: %w", err)
	}
	return result, nil
}

func (repository *SupplyInventoryRepository) ListSupplies(ctx context.Context) ([]domain.Supply, error) {
	rows, err := repository.database.QueryContext(ctx, `SELECT `+supplyColumns+` FROM supplies ORDER BY name,id LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("query supplies: %w", err)
	}
	defer rows.Close()
	return scanSupplies(rows)
}

func (repository *SupplyInventoryRepository) UpdateSupply(ctx context.Context, id string, values domain.SupplyValues, now time.Time) (domain.Supply, error) {
	result, err := scanSupply(repository.database.QueryRowContext(ctx, `
		UPDATE supplies
		SET name=$2,sku=$3,unit=$4,replacement_unit_cost_cents=$5,minimum_quantity=$6,notes=$7,updated_at=GREATEST(updated_at,$8)
		WHERE id=$1
		RETURNING `+supplyColumns,
		id, values.Name, values.SKU, values.Unit, values.ReplacementUnitCostCents, values.MinimumQuantity, values.Notes, now.UTC(),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Supply{}, domain.ErrSupplyNotFound
	}
	if uniqueError(err, "supplies_sku_unique") {
		return domain.Supply{}, domain.ErrSupplySKUConflict
	}
	if err != nil {
		return domain.Supply{}, fmt.Errorf("update supply: %w", err)
	}
	return result, nil
}

func (repository *SupplyInventoryRepository) DeleteSupply(ctx context.Context, id string) error {
	result, err := repository.database.ExecContext(ctx, "DELETE FROM supplies WHERE id=$1", id)
	if fkError(err, "supply_movements_supply_fk") {
		return domain.ErrSupplyHistoryExists
	}
	if err != nil {
		return fmt.Errorf("delete supply: %w", err)
	}
	return requireDeleted(result, domain.ErrSupplyNotFound)
}

func (repository *SupplyInventoryRepository) RecordSupplyMovement(ctx context.Context, supplyID, actorID string, values domain.SupplyMovementValues, createdAt time.Time) (domain.SupplyMovement, error) {
	tx, err := repository.database.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return domain.SupplyMovement{}, fmt.Errorf("begin supply movement transaction: %w", err)
	}
	defer tx.Rollback()

	var currentQuantity string
	if err := tx.QueryRowContext(ctx, "SELECT current_quantity::text FROM supplies WHERE id=$1 FOR UPDATE", supplyID).Scan(&currentQuantity); errors.Is(err, sql.ErrNoRows) {
		return domain.SupplyMovement{}, domain.ErrSupplyNotFound
	} else if err != nil {
		return domain.SupplyMovement{}, fmt.Errorf("lock supply: %w", err)
	}
	current, currentOK := new(big.Rat).SetString(currentQuantity)
	delta, deltaOK := new(big.Rat).SetString(values.Quantity)
	if !currentOK || !deltaOK {
		return domain.SupplyMovement{}, errors.New("invalid exact supply quantity")
	}
	if new(big.Rat).Add(current, delta).Sign() < 0 {
		return domain.SupplyMovement{}, domain.ErrInsufficientStock
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE supplies
		SET current_quantity=current_quantity+CAST($2 AS NUMERIC),updated_at=GREATEST(updated_at,$3)
		WHERE id=$1`, supplyID, values.Quantity, createdAt.UTC()); err != nil {
		return domain.SupplyMovement{}, fmt.Errorf("update supply quantity: %w", err)
	}
	movement, err := scanSupplyMovement(tx.QueryRowContext(ctx, `
		INSERT INTO supply_movements (supply_id,type,quantity,unit_cost_cents,reference_type,reference_id,occurred_at,recorded_by,notes,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING `+supplyMovementColumns,
		supplyID, values.Type, values.Quantity, values.UnitCostCents, values.ReferenceType, values.ReferenceID, values.OccurredAt.UTC(), actorID, values.Notes, createdAt.UTC(),
	))
	if err != nil {
		return domain.SupplyMovement{}, fmt.Errorf("insert supply movement: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.SupplyMovement{}, fmt.Errorf("commit supply movement: %w", err)
	}
	return movement, nil
}

func (repository *SupplyInventoryRepository) ListSupplyMovements(ctx context.Context, supplyID string) ([]domain.SupplyMovement, error) {
	var exists bool
	if err := repository.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM supplies WHERE id=$1)", supplyID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check supply: %w", err)
	}
	if !exists {
		return nil, domain.ErrSupplyNotFound
	}
	rows, err := repository.database.QueryContext(ctx, `SELECT `+supplyMovementColumns+` FROM supply_movements WHERE supply_id=$1 ORDER BY occurred_at DESC,id DESC LIMIT 100`, supplyID)
	if err != nil {
		return nil, fmt.Errorf("query supply movements: %w", err)
	}
	defer rows.Close()
	result := []domain.SupplyMovement{}
	for rows.Next() {
		value, err := scanSupplyMovement(rows)
		if err != nil {
			return nil, fmt.Errorf("scan supply movement: %w", err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supply movements: %w", err)
	}
	return result, nil
}

func (repository *SupplyInventoryRepository) ListLowInventory(ctx context.Context, spoolThresholdG string) (domain.LowInventory, error) {
	supplyRows, err := repository.database.QueryContext(ctx, `SELECT `+supplyColumns+` FROM supplies WHERE minimum_quantity > 0 AND current_quantity <= minimum_quantity ORDER BY (minimum_quantity-current_quantity) DESC,name,id LIMIT 100`)
	if err != nil {
		return domain.LowInventory{}, fmt.Errorf("query low supplies: %w", err)
	}
	supplies, err := scanSupplies(supplyRows)
	supplyRows.Close()
	if err != nil {
		return domain.LowInventory{}, err
	}

	spoolRows, err := repository.database.QueryContext(ctx, `SELECT `+spoolColumns+` FROM material_spools WHERE current_remaining_weight_g IS NOT NULL AND current_remaining_weight_g <= $1 AND status IN ('open','stored','drying') ORDER BY current_remaining_weight_g,code,id LIMIT 100`, spoolThresholdG)
	if err != nil {
		return domain.LowInventory{}, fmt.Errorf("query low spools: %w", err)
	}
	defer spoolRows.Close()
	spools := []domain.Spool{}
	for spoolRows.Next() {
		value, err := scanSpool(spoolRows)
		if err != nil {
			return domain.LowInventory{}, fmt.Errorf("scan low spool: %w", err)
		}
		spools = append(spools, value)
	}
	if err := spoolRows.Err(); err != nil {
		return domain.LowInventory{}, fmt.Errorf("iterate low spools: %w", err)
	}
	return domain.LowInventory{Spools: spools, Supplies: supplies}, nil
}

func scanSupply(row rowScanner) (domain.Supply, error) {
	var value domain.Supply
	var sku sql.NullString
	if err := row.Scan(&value.ID, &value.Name, &sku, &value.Unit, &value.CurrentQuantity, &value.ReplacementUnitCostCents, &value.MinimumQuantity, &value.Notes, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return domain.Supply{}, err
	}
	if sku.Valid {
		value.SKU = &sku.String
	}
	value.CreatedAt = value.CreatedAt.UTC()
	value.UpdatedAt = value.UpdatedAt.UTC()
	return value, nil
}

func scanSupplies(rows *sql.Rows) ([]domain.Supply, error) {
	result := []domain.Supply{}
	for rows.Next() {
		value, err := scanSupply(rows)
		if err != nil {
			return nil, fmt.Errorf("scan supply: %w", err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate supplies: %w", err)
	}
	return result, nil
}

func scanSupplyMovement(row rowScanner) (domain.SupplyMovement, error) {
	var value domain.SupplyMovement
	var unitCost sql.NullInt64
	var referenceType, referenceID sql.NullString
	if err := row.Scan(&value.ID, &value.SupplyID, &value.Type, &value.Quantity, &unitCost, &referenceType, &referenceID, &value.OccurredAt, &value.RecordedBy, &value.Notes, &value.CreatedAt); err != nil {
		return domain.SupplyMovement{}, err
	}
	if unitCost.Valid {
		value.UnitCostCents = &unitCost.Int64
	}
	if referenceType.Valid {
		value.ReferenceType = &referenceType.String
		value.ReferenceID = &referenceID.String
	}
	value.OccurredAt = value.OccurredAt.UTC()
	value.CreatedAt = value.CreatedAt.UTC()
	return value, nil
}
