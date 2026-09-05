package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
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
		_, _ = db.ExecContext(ctx, "TRUNCATE TABLE job_events, print_jobs, catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, printers, workshop_settings, files, sessions, bootstrap_state, users, client_devices")
		_ = db.Close()
	})
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate()=%v", err)
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE TABLE job_events, print_jobs, catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, printers, workshop_settings, files, sessions, bootstrap_state, users, client_devices"); err != nil {
		t.Fatalf("truncate=%v", err)
	}
	var userID, deviceID, itemID, partID, designID, printerID string
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
	repo := NewJobRepository(db)
	now := time.Date(2026, 9, 4, 21, 0, 0, 0, time.UTC)
	actor := domain.Actor{UserID: userID, DeviceID: deviceID}
	values := domain.Values{Code: "JOB-I-1", CatalogItemID: itemID, DesignVersionID: designID, PrinterID: printerID, Purpose: domain.PurposeInternal, PlannedQuantity: 2}
	job, err := repo.Create(ctx, values, actor, now)
	if err != nil || job.OrderItemID != nil || job.Purpose != domain.PurposeInternal {
		t.Fatalf("Create()=%#v,%v", job, err)
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
}
