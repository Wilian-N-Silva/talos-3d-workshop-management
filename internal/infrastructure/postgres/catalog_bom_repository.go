package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
	domaininventory "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
)

const catalogBOMColumns = `id, catalog_item_id, supply_id, quantity_per_unit::text, waste_percent::text, notes, created_at, updated_at`
const qualifiedCatalogBOMColumns = `b.id, b.catalog_item_id, b.supply_id, b.quantity_per_unit::text, b.waste_percent::text, b.notes, b.created_at, b.updated_at`

type CatalogBOMRepository struct {
	database *sql.DB
}

func NewCatalogBOMRepository(database *sql.DB) *CatalogBOMRepository {
	return &CatalogBOMRepository{database: database}
}

func (repository *CatalogBOMRepository) CreateBOMItem(ctx context.Context, catalogItemID string, values domaincatalog.BOMValues, now time.Time) (domaincatalog.BOMItem, error) {
	item, err := scanCatalogBOMItem(repository.database.QueryRowContext(ctx, `
		INSERT INTO catalog_bom_items (catalog_item_id,supply_id,quantity_per_unit,waste_percent,notes,created_at,updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$6)
		RETURNING `+catalogBOMColumns,
		catalogItemID, values.SupplyID, values.QuantityPerUnit, values.WastePercent, values.Notes, now.UTC(),
	))
	if fkError(err, "catalog_bom_catalog_item_fk") {
		return domaincatalog.BOMItem{}, domaincatalog.ErrItemNotFound
	}
	if fkError(err, "catalog_bom_supply_fk") {
		return domaincatalog.BOMItem{}, domaininventory.ErrSupplyNotFound
	}
	if uniqueError(err, "catalog_bom_item_supply_unique") {
		return domaincatalog.BOMItem{}, domaincatalog.ErrBOMSupplyConflict
	}
	if err != nil {
		return domaincatalog.BOMItem{}, fmt.Errorf("insert catalog BOM item: %w", err)
	}
	return item, nil
}

func (repository *CatalogBOMRepository) FindBOMItem(ctx context.Context, catalogItemID, bomItemID string) (domaincatalog.BOMItem, error) {
	item, err := scanCatalogBOMItem(repository.database.QueryRowContext(ctx, `SELECT `+catalogBOMColumns+` FROM catalog_bom_items WHERE catalog_item_id=$1 AND id=$2`, catalogItemID, bomItemID))
	if errors.Is(err, sql.ErrNoRows) {
		return domaincatalog.BOMItem{}, domaincatalog.ErrBOMItemNotFound
	}
	if err != nil {
		return domaincatalog.BOMItem{}, fmt.Errorf("find catalog BOM item: %w", err)
	}
	return item, nil
}

func (repository *CatalogBOMRepository) ListBOMCostInputs(ctx context.Context, catalogItemID string) ([]domaincatalog.BOMCostInput, error) {
	exists, err := repository.catalogItemExists(ctx, catalogItemID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domaincatalog.ErrItemNotFound
	}
	rows, err := repository.database.QueryContext(ctx, `
		SELECT `+qualifiedCatalogBOMColumns+`,s.name,s.unit,s.replacement_unit_cost_cents
		FROM catalog_bom_items b
		JOIN supplies s ON s.id=b.supply_id
		WHERE b.catalog_item_id=$1
		ORDER BY s.name,b.id
		LIMIT 100`, catalogItemID)
	if err != nil {
		return nil, fmt.Errorf("query catalog BOM: %w", err)
	}
	defer rows.Close()
	result := []domaincatalog.BOMCostInput{}
	for rows.Next() {
		var value domaincatalog.BOMCostInput
		if err := scanCatalogBOMCostInput(rows, &value); err != nil {
			return nil, fmt.Errorf("scan catalog BOM: %w", err)
		}
		result = append(result, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog BOM: %w", err)
	}
	return result, nil
}

func (repository *CatalogBOMRepository) UpdateBOMItem(ctx context.Context, catalogItemID, bomItemID string, values domaincatalog.BOMValues, now time.Time) (domaincatalog.BOMItem, error) {
	item, err := scanCatalogBOMItem(repository.database.QueryRowContext(ctx, `
		UPDATE catalog_bom_items
		SET supply_id=$3,quantity_per_unit=$4,waste_percent=$5,notes=$6,updated_at=GREATEST(updated_at,$7)
		WHERE catalog_item_id=$1 AND id=$2
		RETURNING `+catalogBOMColumns,
		catalogItemID, bomItemID, values.SupplyID, values.QuantityPerUnit, values.WastePercent, values.Notes, now.UTC(),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domaincatalog.BOMItem{}, domaincatalog.ErrBOMItemNotFound
	}
	if fkError(err, "catalog_bom_supply_fk") {
		return domaincatalog.BOMItem{}, domaininventory.ErrSupplyNotFound
	}
	if uniqueError(err, "catalog_bom_item_supply_unique") {
		return domaincatalog.BOMItem{}, domaincatalog.ErrBOMSupplyConflict
	}
	if err != nil {
		return domaincatalog.BOMItem{}, fmt.Errorf("update catalog BOM item: %w", err)
	}
	return item, nil
}

func (repository *CatalogBOMRepository) DeleteBOMItem(ctx context.Context, catalogItemID, bomItemID string) error {
	result, err := repository.database.ExecContext(ctx, "DELETE FROM catalog_bom_items WHERE catalog_item_id=$1 AND id=$2", catalogItemID, bomItemID)
	if err != nil {
		return fmt.Errorf("delete catalog BOM item: %w", err)
	}
	return requireDeleted(result, domaincatalog.ErrBOMItemNotFound)
}

func (repository *CatalogBOMRepository) catalogItemExists(ctx context.Context, catalogItemID string) (bool, error) {
	var exists bool
	if err := repository.database.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM catalog_items WHERE id=$1)", catalogItemID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check catalog item: %w", err)
	}
	return exists, nil
}

func scanCatalogBOMItem(row rowScanner) (domaincatalog.BOMItem, error) {
	var item domaincatalog.BOMItem
	if err := row.Scan(&item.ID, &item.CatalogItemID, &item.SupplyID, &item.QuantityPerUnit, &item.WastePercent, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return domaincatalog.BOMItem{}, err
	}
	item.CreatedAt = item.CreatedAt.UTC()
	item.UpdatedAt = item.UpdatedAt.UTC()
	return item, nil
}

func scanCatalogBOMCostInput(row rowScanner, value *domaincatalog.BOMCostInput) error {
	if err := row.Scan(
		&value.Item.ID, &value.Item.CatalogItemID, &value.Item.SupplyID,
		&value.Item.QuantityPerUnit, &value.Item.WastePercent, &value.Item.Notes,
		&value.Item.CreatedAt, &value.Item.UpdatedAt,
		&value.SupplyName, &value.SupplyUnit, &value.ReplacementUnitCostCents,
	); err != nil {
		return err
	}
	value.Item.CreatedAt = value.Item.CreatedAt.UTC()
	value.Item.UpdatedAt = value.Item.UpdatedAt.UTC()
	return nil
}
