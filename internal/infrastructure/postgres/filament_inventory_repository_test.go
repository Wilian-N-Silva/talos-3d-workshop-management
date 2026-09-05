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

func TestFilamentInventoryRepositoryAgainstPostgreSQL(t *testing.T) {
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
	user, err := NewUserRepository(database).Create(ctx, domainauth.CreateUserParams{Name: "Operator", EmailOrUsername: "inventory-operator", PasswordHash: "$argon2id$test", Status: domainauth.UserStatusActive, Role: domainauth.RoleOperator})
	if err != nil {
		t.Fatalf("create user=%v", err)
	}
	repository := NewFilamentInventoryRepository(database)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	material, err := repository.CreateMaterial(ctx, domain.MaterialValues{Manufacturer: "Voolt3D", Name: "Velvet White", MaterialType: "PLA", ColorName: "White", NominalDensity: "1.240000", DefaultReplacementCostPerKgCents: 12990}, now)
	if err != nil || material.ID == "" || material.NominalDensity != "1.240000" {
		t.Fatalf("CreateMaterial()=%#v,%v", material, err)
	}
	found, err := repository.FindMaterial(ctx, material.ID)
	if err != nil || found.ID != material.ID {
		t.Fatalf("FindMaterial()=%#v,%v", found, err)
	}
	materials, err := repository.ListMaterials(ctx)
	if err != nil || len(materials) != 1 {
		t.Fatalf("ListMaterials()=%#v,%v", materials, err)
	}
	updated, err := repository.UpdateMaterial(ctx, material.ID, domain.MaterialValues{Manufacturer: "Voolt3D", Name: "Velvet Branco", MaterialType: "PLA", NominalDensity: "1.24", DefaultReplacementCostPerKgCents: 13990}, now.Add(time.Hour))
	if err != nil || updated.Name != "Velvet Branco" || updated.DefaultReplacementCostPerKgCents != 13990 {
		t.Fatalf("UpdateMaterial()=%#v,%v", updated, err)
	}
	spool, err := repository.CreateSpool(ctx, domain.SpoolValues{Code: "PLA-001", MaterialID: material.ID, NominalNetWeightG: "1000", TareWeightG: "250", PurchaseCostCents: 9990, ReplacementCostPerKgCents: 13990, StorageLocation: "Shelf A", Status: domain.SpoolOpen}, now)
	if err != nil || spool.ID == "" {
		t.Fatalf("CreateSpool()=%#v,%v", spool, err)
	}
	if _, err := repository.CreateSpool(ctx, domain.SpoolValues{Code: "pla-001", MaterialID: material.ID, NominalNetWeightG: "1000", TareWeightG: "250", Status: domain.SpoolSealed}, now); !errors.Is(err, domain.ErrSpoolCodeConflict) {
		t.Fatalf("duplicate code=%v", err)
	}
	measurement, err := repository.RecordMeasurement(ctx, spool.ID, user.ID, domain.MeasurementValues{MeasuredAt: now.Add(2 * time.Hour), GrossWeightG: "845.500", Source: domain.MeasurementManual, Notes: "bench"}, now.Add(2*time.Hour))
	if err != nil || measurement.DerivedRemainingWeightG != "595.500" {
		t.Fatalf("RecordMeasurement()=%#v,%v", measurement, err)
	}
	measuredSpool, err := repository.FindSpool(ctx, spool.ID)
	if err != nil || measuredSpool.CurrentRemainingWeightG == nil || *measuredSpool.CurrentRemainingWeightG != "595.500" || measuredSpool.LastWeighedAt == nil || !measuredSpool.LastWeighedAt.Equal(now.Add(2*time.Hour)) {
		t.Fatalf("measurement cache=%#v,%v", measuredSpool, err)
	}
	olderMeasurement, err := repository.RecordMeasurement(ctx, spool.ID, user.ID, domain.MeasurementValues{MeasuredAt: now.Add(time.Hour), GrossWeightG: "900", Source: domain.MeasurementImported}, now.Add(3*time.Hour))
	if err != nil || olderMeasurement.DerivedRemainingWeightG != "650.000" {
		t.Fatalf("older RecordMeasurement()=%#v,%v", olderMeasurement, err)
	}
	measuredSpool, err = repository.FindSpool(ctx, spool.ID)
	if err != nil || measuredSpool.CurrentRemainingWeightG == nil || *measuredSpool.CurrentRemainingWeightG != "595.500" || measuredSpool.LastWeighedAt == nil || !measuredSpool.LastWeighedAt.Equal(now.Add(2*time.Hour)) {
		t.Fatalf("cache after older measurement=%#v,%v", measuredSpool, err)
	}
	if _, err := repository.RecordMeasurement(ctx, spool.ID, user.ID, domain.MeasurementValues{MeasuredAt: now.Add(3 * time.Hour), GrossWeightG: "200", Source: domain.MeasurementManual}, now.Add(3*time.Hour)); !errors.Is(err, domain.ErrMeasurementBelowTare) {
		t.Fatalf("below tare error=%v", err)
	}
	history, err := repository.ListMeasurements(ctx, spool.ID)
	if err != nil || len(history) != 2 || history[0].ID != measurement.ID || history[1].ID != olderMeasurement.ID {
		t.Fatalf("ListMeasurements()=%#v,%v", history, err)
	}
	if err := repository.DeleteSpool(ctx, spool.ID); !errors.Is(err, domain.ErrInventoryHistoryExists) {
		t.Fatalf("DeleteSpool(history)=%v", err)
	}
	if err := repository.DeleteMaterial(ctx, material.ID); !errors.Is(err, domain.ErrInventoryHistoryExists) {
		t.Fatalf("DeleteMaterial(spool)=%v", err)
	}
	emptySpool, err := repository.CreateSpool(ctx, domain.SpoolValues{Code: "PLA-EMPTY", MaterialID: material.ID, NominalNetWeightG: "1000", TareWeightG: "250", Status: domain.SpoolRetired}, now)
	if err != nil {
		t.Fatalf("create empty spool=%v", err)
	}
	if err := repository.DeleteSpool(ctx, emptySpool.ID); err != nil {
		t.Fatalf("DeleteSpool(empty)=%v", err)
	}
}
