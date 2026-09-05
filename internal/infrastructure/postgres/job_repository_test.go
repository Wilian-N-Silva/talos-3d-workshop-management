package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	domainenergy "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/energy"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
	domainlabor "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/labor"
)

func TestJobRepositoryNonCommercialLifecycleAgainstPostgreSQL(t *testing.T) {
	url := os.Getenv("TALOS_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TALOS_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := Open(ctx, testConfig(url))
	if err != nil {
		t.Fatalf("Open()=%v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, "TRUNCATE TABLE job_labor_entries, labor_rates, energy_measurements, print_job_material_usage, job_events, print_jobs, catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, printers, workshop_settings, files, sessions, bootstrap_state, users, client_devices")
		_ = db.Close()
	})
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate()=%v", err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE job_labor_entries, labor_rates, energy_measurements, print_job_material_usage, job_events, print_jobs, catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, printers, workshop_settings, files, sessions, bootstrap_state, users, client_devices"); err != nil {
		t.Fatalf("truncate=%v", err)
	}
	var userID, deviceID, itemID, partID, designID, printerID, materialID, spoolID string
	if err := db.QueryRowContext(ctx, "INSERT INTO users(name,email_or_username,password_hash,status,role) VALUES('Operator','operator','$argon2id$test','active','operator') RETURNING id").Scan(&userID); err != nil {
		t.Fatalf("user=%v", err)
	}
	if err := db.QueryRowContext(ctx, "INSERT INTO client_devices(display_name,os,app_version) VALUES('Bench','windows','1') RETURNING id").Scan(&deviceID); err != nil {
		t.Fatalf("device=%v", err)
	}
	if err := db.QueryRowContext(ctx, "INSERT INTO catalog_items(name,purpose,sellable,tags,status) VALUES('Fixture','internal',false,'[]','active') RETURNING id").Scan(&itemID); err != nil {
		t.Fatalf("item=%v", err)
	}
	if err := db.QueryRowContext(ctx, "INSERT INTO catalog_parts(catalog_item_id,name) VALUES($1,'Body') RETURNING id", itemID).Scan(&partID); err != nil {
		t.Fatalf("part=%v", err)
	}
	if err := db.QueryRowContext(ctx, "INSERT INTO design_versions(catalog_part_id,version,created_by) VALUES($1,'v1',$2) RETURNING id", partID, userID).Scan(&designID); err != nil {
		t.Fatalf("design=%v", err)
	}
	if err := db.QueryRowContext(ctx, "INSERT INTO printers(name,manufacturer,model,nozzle_diameter,acquisition_cost_cents,residual_value_cents,useful_life_hours,maintenance_reserve_per_hour_cents) VALUES('A1','Bambu','A1',0.4,100000,10000,5000,20) RETURNING id").Scan(&printerID); err != nil {
		t.Fatalf("printer=%v", err)
	}
	if err := db.QueryRowContext(ctx, "INSERT INTO materials(manufacturer,name,material_type,color_name,nominal_density,default_replacement_cost_per_kg_cents) VALUES('Talos','PLA Black','PLA','Black',1.24,12000) RETURNING id").Scan(&materialID); err != nil {
		t.Fatalf("material=%v", err)
	}
	if err := db.QueryRowContext(ctx, "INSERT INTO material_spools(code,material_id,nominal_net_weight_g,tare_weight_g,purchase_cost_cents,replacement_cost_per_kg_cents,status) VALUES('SPOOL-JOB-1',$1,1000,250,10000,12000,'open') RETURNING id", materialID).Scan(&spoolID); err != nil {
		t.Fatalf("spool=%v", err)
	}
	repo := NewJobRepository(db)
	now := time.Date(2026, 9, 4, 21, 0, 0, 0, time.UTC)
	actor := domain.Actor{UserID: userID, DeviceID: deviceID}
	values := domain.Values{Code: "JOB-I-1", CatalogItemID: itemID, DesignVersionID: designID, PrinterID: printerID, Purpose: domain.PurposeInternal, PlannedQuantity: 2}
	job, err := repo.Create(ctx, values, actor, now)
	if err != nil || job.OrderItemID != nil || job.Purpose != domain.PurposeInternal {
		t.Fatalf("Create()=%#v,%v", job, err)
	}
	energyRepository := NewEnergyRepository(db)
	startKWh, endKWh, measuredKWh := "120.125", "121.375", "1.25"
	energyMeasurement, err := energyRepository.Create(ctx, job.ID, userID, domainenergy.Values{Source: domainenergy.SourceManualMeter, MeterStartKWh: &startKWh, MeterEndKWh: &endKWh, MeasuredKWh: &measuredKWh, EnergyRateCentsPerKWh: 95, OccurredAt: now, Notes: "bench meter"})
	if err != nil || energyMeasurement.MeasuredKWh == nil || *energyMeasurement.MeasuredKWh != "1.25" || energyMeasurement.EnergyRateCentsPerKWh != 95 || energyMeasurement.RecordedBy != userID {
		t.Fatalf("Create energy measurement=%#v,%v", energyMeasurement, err)
	}
	energyMeasurements, err := energyRepository.List(ctx, job.ID)
	if err != nil || len(energyMeasurements) != 1 || energyMeasurements[0].Source != domainenergy.SourceManualMeter {
		t.Fatalf("List energy measurements=%#v,%v", energyMeasurements, err)
	}
	laborRepository := NewLaborRepository(db)
	laborRate, err := laborRepository.CreateRate(ctx, domainlabor.RateValues{Name: "Internal setup", ActivityType: domainlabor.ActivitySetup, CostHourlyRateCents: 6000, Active: true}, now)
	if err != nil || laborRate.CostHourlyRateCents != 6000 {
		t.Fatalf("Create labor rate=%#v,%v", laborRate, err)
	}
	laborEntry, err := laborRepository.CreateEntry(ctx, job.ID, userID, domainlabor.EntryValues{LaborRateID: laborRate.ID, Minutes: 15, OccurredAt: now, Notes: "fixture setup"}, now)
	if err != nil || laborEntry.ActivityType != domainlabor.ActivitySetup || laborEntry.InternalHourlyRateCents != 6000 || laborEntry.RecordedBy != userID {
		t.Fatalf("Create labor entry=%#v,%v", laborEntry, err)
	}
	laborRate, err = laborRepository.UpdateRate(ctx, laborRate.ID, domainlabor.RateValues{Name: "Internal setup", ActivityType: domainlabor.ActivitySetup, CostHourlyRateCents: 9000, Active: true}, now.Add(time.Second))
	if err != nil || laborRate.CostHourlyRateCents != 9000 {
		t.Fatalf("Update labor rate=%#v,%v", laborRate, err)
	}
	laborEntries, err := laborRepository.ListEntries(ctx, job.ID)
	if err != nil || len(laborEntries) != 1 || laborEntries[0].InternalHourlyRateCents != 6000 {
		t.Fatalf("List labor entries=%#v,%v", laborEntries, err)
	}
	historicalCost, replacementCost := int64(100), int64(120)
	modelUsage, err := repo.CreateMaterialUsage(ctx, job.ID, domain.MaterialUsageValues{SpoolID: spoolID, Role: domain.MaterialRoleModel, PlannedGrams: "10.125", MeasurementSource: domain.SourceSlicer, HistoricalMaterialCostCents: &historicalCost, ReplacementMaterialCostCents: &replacementCost}, now)
	if err != nil || modelUsage.MaterialID != materialID || modelUsage.MeasurementSource != domain.SourceSlicer || modelUsage.HistoricalMaterialCostCents == nil || *modelUsage.HistoricalMaterialCostCents != historicalCost {
		t.Fatalf("CreateMaterialUsage(model)=%#v,%v", modelUsage, err)
	}
	supportUsage, err := repo.CreateMaterialUsage(ctx, job.ID, domain.MaterialUsageValues{SpoolID: spoolID, Role: domain.MaterialRoleSupport, PlannedGrams: "3.875", MeasurementSource: domain.SourceEstimated}, now)
	if err != nil || supportUsage.Role != domain.MaterialRoleSupport {
		t.Fatalf("CreateMaterialUsage(support)=%#v,%v", supportUsage, err)
	}
	if _, err := repo.CreateMaterialUsage(ctx, job.ID, domain.MaterialUsageValues{SpoolID: spoolID, Role: domain.MaterialRoleModel, PlannedGrams: "1", MeasurementSource: domain.SourceManual}, now); !errors.Is(err, domain.ErrMaterialUsageConflict) {
		t.Fatalf("duplicate CreateMaterialUsage() error=%v, want conflict", err)
	}
	actual := "9.5"
	changedCost := int64(999)
	modelUsage, err = repo.UpdateMaterialUsage(ctx, job.ID, modelUsage.ID, domain.MaterialUsageValues{SpoolID: spoolID, Role: domain.MaterialRoleModel, PlannedGrams: "10.125", ActualGrams: &actual, MeasurementSource: domain.SourceSpoolWeightDelta, HistoricalMaterialCostCents: &changedCost, ReplacementMaterialCostCents: &changedCost}, now.Add(time.Second))
	if err != nil || modelUsage.ActualGrams == nil || *modelUsage.ActualGrams != actual || modelUsage.MeasurementSource != domain.SourceSpoolWeightDelta || modelUsage.HistoricalMaterialCostCents == nil || *modelUsage.HistoricalMaterialCostCents != historicalCost || modelUsage.ReplacementMaterialCostCents == nil || *modelUsage.ReplacementMaterialCostCents != replacementCost {
		t.Fatalf("UpdateMaterialUsage()=%#v,%v", modelUsage, err)
	}
	usages, err := repo.ListMaterialUsage(ctx, job.ID)
	if err != nil || len(usages) != 2 {
		t.Fatalf("ListMaterialUsage()=%#v,%v", usages, err)
	}
	if err := repo.DeleteMaterialUsage(ctx, job.ID, supportUsage.ID); err != nil {
		t.Fatalf("DeleteMaterialUsage()=%v", err)
	}
	steps := []struct {
		from, to domain.Status
		event    domain.EventType
	}{{domain.StatusDraft, domain.StatusPrepared, domain.EventPrepared}, {domain.StatusPrepared, domain.StatusPrinting, domain.EventPrintingStartedManual}, {domain.StatusPrinting, domain.StatusAwaitingReview, domain.EventFinishedManual}}
	for index, step := range steps {
		job, err = repo.Transition(ctx, job.ID, step.from, domain.TransitionValues{Status: step.to}, step.event, actor, now.Add(time.Duration(index+1)*time.Minute))
		if err != nil || job.Status != step.to {
			t.Fatalf("transition %s=%#v,%v", step.to, job, err)
		}
	}
	job, err = repo.Review(ctx, job.ID, domain.ReviewValues{QualityStatus: domain.QualityPartial, GoodQuantity: 1, ScrapQuantity: 1, ResultNotes: "one rejected"}, actor, now.Add(4*time.Minute))
	if err != nil || job.Status != domain.StatusCompleted || job.QualityStatus != domain.QualityPartial || job.GoodQuantity != 1 || job.CompletedAt == nil {
		t.Fatalf("Review()=%#v,%v", job, err)
	}
	events, err := repo.ListEvents(ctx, job.ID)
	if err != nil || len(events) != 5 || events[0].Type != domain.EventCreated || events[4].Type != domain.EventReviewed || events[4].ActorUserID != userID || events[4].SourceDeviceID != deviceID {
		t.Fatalf("ListEvents()=%#v,%v", events, err)
	}
	if _, err := repo.UpdateMaterialUsage(ctx, job.ID, modelUsage.ID, domain.MaterialUsageValues{SpoolID: spoolID, Role: domain.MaterialRoleModel, PlannedGrams: "10", MeasurementSource: domain.SourceManual}, now.Add(5*time.Minute)); !errors.Is(err, domain.ErrJobNotEditable) {
		t.Fatalf("terminal UpdateMaterialUsage() error=%v, want not editable", err)
	}
}
