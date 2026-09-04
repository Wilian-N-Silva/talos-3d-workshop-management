package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
	domaininventory "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
)

func TestCatalogBOMRepositoryAgainstPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("TALOS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TALOS_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	database, err := Open(ctx, testConfig(databaseURL))
	if err != nil {
		t.Fatalf("Open()=%v", err)
	}
	t.Cleanup(func() {
		_, _ = database.ExecContext(ctx, "TRUNCATE TABLE catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, workshop_settings, files, sessions, bootstrap_state, users, client_devices")
		_ = database.Close()
	})
	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate()=%v", err)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, workshop_settings, files, sessions, bootstrap_state, users, client_devices"); err != nil {
		t.Fatalf("truncate=%v", err)
	}
	now := time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC)
	catalogItem, err := NewCatalogItemRepository(database).Create(ctx, domaincatalog.Values{Name: "Smart tag", Purpose: domaincatalog.PurposeProduct, Sellable: true, Tags: []string{}, Status: domaincatalog.StatusActive}, now)
	if err != nil {
		t.Fatalf("create catalog item=%v", err)
	}
	supply, err := NewSupplyInventoryRepository(database).CreateSupply(ctx, domaininventory.SupplyValues{Name: "NFC", Unit: "unit", ReplacementUnitCostCents: 75, MinimumQuantity: "10"}, now)
	if err != nil {
		t.Fatalf("create supply=%v", err)
	}
	repository := NewCatalogBOMRepository(database)
	bomItem, err := repository.CreateBOMItem(ctx, catalogItem.ID, domaincatalog.BOMValues{SupplyID: supply.ID, QuantityPerUnit: "1.000000", WastePercent: "10.0000", Notes: "tag"}, now)
	if err != nil || bomItem.ID == "" || bomItem.QuantityPerUnit != "1.000000" {
		t.Fatalf("CreateBOMItem()=%#v,%v", bomItem, err)
	}
	if _, err := repository.CreateBOMItem(ctx, catalogItem.ID, domaincatalog.BOMValues{SupplyID: supply.ID, QuantityPerUnit: "2", WastePercent: "0"}, now); !errors.Is(err, domaincatalog.ErrBOMSupplyConflict) {
		t.Fatalf("duplicate supply=%v", err)
	}
	inputs, err := repository.ListBOMCostInputs(ctx, catalogItem.ID)
	if err != nil || len(inputs) != 1 || inputs[0].SupplyName != "NFC" || inputs[0].ReplacementUnitCostCents != 75 {
		t.Fatalf("ListBOMCostInputs()=%#v,%v", inputs, err)
	}
	updated, err := repository.UpdateBOMItem(ctx, catalogItem.ID, bomItem.ID, domaincatalog.BOMValues{SupplyID: supply.ID, QuantityPerUnit: "2.000000", WastePercent: "5.0000", Notes: "two tags"}, now.Add(time.Hour))
	if err != nil || updated.QuantityPerUnit != "2.000000" || updated.WastePercent != "5.0000" {
		t.Fatalf("UpdateBOMItem()=%#v,%v", updated, err)
	}
	if err := NewSupplyInventoryRepository(database).DeleteSupply(ctx, supply.ID); !errors.Is(err, domaininventory.ErrSupplyInUse) {
		t.Fatalf("DeleteSupply(in use)=%v", err)
	}
	if err := repository.DeleteBOMItem(ctx, catalogItem.ID, bomItem.ID); err != nil {
		t.Fatalf("DeleteBOMItem()=%v", err)
	}
	if err := NewSupplyInventoryRepository(database).DeleteSupply(ctx, supply.ID); err != nil {
		t.Fatalf("DeleteSupply(after BOM)=%v", err)
	}
	if _, err := repository.ListBOMCostInputs(ctx, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"); !errors.Is(err, domaincatalog.ErrItemNotFound) {
		t.Fatalf("missing catalog item=%v", err)
	}
}
