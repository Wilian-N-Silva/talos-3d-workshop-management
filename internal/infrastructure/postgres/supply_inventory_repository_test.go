package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
)

func TestSupplyInventoryRepositoryAgainstPostgreSQL(t *testing.T) {
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
		_, _ = database.ExecContext(ctx, "TRUNCATE TABLE maintenance_events, job_labor_entries, labor_rates, energy_measurements, print_job_material_usage, job_events, print_jobs, catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, workshop_settings, files, sessions, bootstrap_state, users, client_devices")
		_ = database.Close()
	})
	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate()=%v", err)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE maintenance_events, job_labor_entries, labor_rates, energy_measurements, print_job_material_usage, job_events, print_jobs, catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, workshop_settings, files, sessions, bootstrap_state, users, client_devices"); err != nil {
		t.Fatalf("truncate=%v", err)
	}
	user, err := NewUserRepository(database).Create(ctx, domainauth.CreateUserParams{Name: "Operator", EmailOrUsername: "supply-operator", PasswordHash: "$argon2id$test", Status: domainauth.UserStatusActive, Role: domainauth.RoleOperator})
	if err != nil {
		t.Fatalf("create user=%v", err)
	}

	repository := NewSupplyInventoryRepository(database)
	now := time.Date(2026, 9, 4, 15, 0, 0, 0, time.UTC)
	sku := "NFC-01"
	supply, err := repository.CreateSupply(ctx, domain.SupplyValues{Name: "NFC tag", SKU: &sku, Unit: "unit", ReplacementUnitCostCents: 75, MinimumQuantity: "10"}, now)
	if err != nil || supply.ID == "" || supply.CurrentQuantity != "0.000000" {
		t.Fatalf("CreateSupply()=%#v,%v", supply, err)
	}
	duplicateSKU := "nfc-01"
	if _, err := repository.CreateSupply(ctx, domain.SupplyValues{Name: "Duplicate", SKU: &duplicateSKU, Unit: "unit", MinimumQuantity: "0"}, now); !errors.Is(err, domain.ErrSupplySKUConflict) {
		t.Fatalf("duplicate SKU=%v", err)
	}
	listed, err := repository.ListSupplies(ctx)
	if err != nil || len(listed) != 1 || listed[0].ID != supply.ID {
		t.Fatalf("ListSupplies()=%#v,%v", listed, err)
	}
	updated, err := repository.UpdateSupply(ctx, supply.ID, domain.SupplyValues{Name: "NFC sticker", SKU: &sku, Unit: "unit", ReplacementUnitCostCents: 80, MinimumQuantity: "10"}, now.Add(time.Minute))
	if err != nil || updated.Name != "NFC sticker" || updated.ReplacementUnitCostCents != 80 || updated.CurrentQuantity != "0.000000" {
		t.Fatalf("UpdateSupply()=%#v,%v", updated, err)
	}
	purchase, err := repository.RecordSupplyMovement(ctx, supply.ID, user.ID, domain.SupplyMovementValues{Type: domain.SupplyPurchase, Quantity: "20", UnitCostCents: int64Pointer(60), OccurredAt: now}, now)
	if err != nil || purchase.Quantity != "20.000000" {
		t.Fatalf("purchase=%#v,%v", purchase, err)
	}
	consume, err := repository.RecordSupplyMovement(ctx, supply.ID, user.ID, domain.SupplyMovementValues{Type: domain.SupplyConsume, Quantity: "-4.5", OccurredAt: now.Add(time.Hour)}, now.Add(time.Hour))
	if err != nil || consume.Quantity != "-4.500000" {
		t.Fatalf("consume=%#v,%v", consume, err)
	}
	stocked, err := repository.FindSupply(ctx, supply.ID)
	if err != nil || stocked.CurrentQuantity != "15.500000" {
		t.Fatalf("stock after movements=%#v,%v", stocked, err)
	}
	if _, err := repository.RecordSupplyMovement(ctx, supply.ID, user.ID, domain.SupplyMovementValues{Type: domain.SupplyConsume, Quantity: "-20", OccurredAt: now.Add(2 * time.Hour)}, now.Add(2*time.Hour)); !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("negative stock movement=%v", err)
	}
	history, err := repository.ListSupplyMovements(ctx, supply.ID)
	if err != nil || len(history) != 2 || history[0].ID != consume.ID || history[1].ID != purchase.ID {
		t.Fatalf("movement history=%#v,%v", history, err)
	}
	if _, err := repository.RecordSupplyMovement(ctx, supply.ID, user.ID, domain.SupplyMovementValues{Type: domain.SupplyAdjustment, Quantity: "-6", OccurredAt: now.Add(3 * time.Hour)}, now.Add(3*time.Hour)); err != nil {
		t.Fatalf("adjustment=%v", err)
	}
	if _, err := repository.RecordSupplyMovement(ctx, supply.ID, user.ID, domain.SupplyMovementValues{Type: domain.SupplyReturn, Quantity: "1", OccurredAt: now.Add(4 * time.Hour)}, now.Add(4*time.Hour)); err != nil {
		t.Fatalf("return=%v", err)
	}
	if _, err := repository.RecordSupplyMovement(ctx, supply.ID, user.ID, domain.SupplyMovementValues{Type: domain.SupplyDiscard, Quantity: "-1", OccurredAt: now.Add(5 * time.Hour)}, now.Add(5*time.Hour)); err != nil {
		t.Fatalf("discard=%v", err)
	}

	filamentRepository := NewFilamentInventoryRepository(database)
	material, err := filamentRepository.CreateMaterial(ctx, domain.MaterialValues{Manufacturer: "Maker", Name: "PLA", MaterialType: "PLA", NominalDensity: "1.24"}, now)
	if err != nil {
		t.Fatalf("create material=%v", err)
	}
	spool, err := filamentRepository.CreateSpool(ctx, domain.SpoolValues{Code: "LOW-PLA", MaterialID: material.ID, NominalNetWeightG: "1000", TareWeightG: "250", Status: domain.SpoolOpen}, now)
	if err != nil {
		t.Fatalf("create spool=%v", err)
	}
	if _, err := filamentRepository.RecordMeasurement(ctx, spool.ID, user.ID, domain.MeasurementValues{MeasuredAt: now, GrossWeightG: "300", Source: domain.MeasurementManual}, now); err != nil {
		t.Fatalf("record low spool=%v", err)
	}
	low, err := repository.ListLowInventory(ctx, "100")
	if err != nil || len(low.Supplies) != 1 || low.Supplies[0].ID != supply.ID || len(low.Spools) != 1 || low.Spools[0].ID != spool.ID {
		t.Fatalf("ListLowInventory()=%#v,%v", low, err)
	}
	if err := repository.DeleteSupply(ctx, supply.ID); !errors.Is(err, domain.ErrSupplyHistoryExists) {
		t.Fatalf("DeleteSupply(history)=%v", err)
	}
	empty, err := repository.CreateSupply(ctx, domain.SupplyValues{Name: "Empty", Unit: "unit", MinimumQuantity: "0"}, now)
	if err != nil {
		t.Fatalf("create empty supply=%v", err)
	}
	if err := repository.DeleteSupply(ctx, empty.ID); err != nil {
		t.Fatalf("DeleteSupply(empty)=%v", err)
	}
}

func int64Pointer(value int64) *int64 { return &value }
