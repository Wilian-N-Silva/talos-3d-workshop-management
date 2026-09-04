package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	domainmaintenance "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/maintenance"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/printers"
)

func TestPrinterRepositoryAgainstPostgreSQL(t *testing.T) {
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
		_, _ = database.ExecContext(ctx, "TRUNCATE TABLE maintenance_events, job_labor_entries, labor_rates, energy_measurements, print_job_material_usage, job_events, print_jobs, printers")
		_ = database.Close()
	})
	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate()=%v", err)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE maintenance_events, job_labor_entries, labor_rates, energy_measurements, print_job_material_usage, job_events, print_jobs, printers"); err != nil {
		t.Fatalf("truncate=%v", err)
	}
	now := time.Date(2026, 9, 4, 20, 0, 0, 0, time.UTC)
	repository := NewPrinterRepository(database)
	values := domain.Values{Name: "A1 Mini", Manufacturer: "Bambu Lab", Model: "A1 Mini", NozzleDiameter: "0.400", Location: "Print room", AcquisitionCostCents: 180000, ResidualValueCents: 30000, UsefulLifeHours: "5000.00", MaintenanceReservePerHourCents: 25, Status: domain.StatusActive, Notes: "Primary"}
	created, err := repository.Create(ctx, values, now)
	if err != nil || created.ID == "" || created.NozzleDiameter != "0.400" || created.UsefulLifeHours != "5000.00" {
		t.Fatalf("Create()=%#v,%v", created, err)
	}
	if _, err := repository.Create(ctx, func() domain.Values { duplicate := values; duplicate.Name = "a1 MINI"; return duplicate }(), now); !errors.Is(err, domain.ErrPrinterNameConflict) {
		t.Fatalf("duplicate name=%v", err)
	}
	listed, err := repository.List(ctx)
	if err != nil || len(listed) != 1 {
		t.Fatalf("List()=%#v,%v", listed, err)
	}
	values.Status = domain.StatusMaintenance
	values.Location = "Service bench"
	updated, err := repository.Update(ctx, created.ID, values, now.Add(time.Hour))
	if err != nil || updated.Status != domain.StatusMaintenance || updated.Location != "Service bench" {
		t.Fatalf("Update()=%#v,%v", updated, err)
	}
	if err := repository.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete()=%v", err)
	}
	if _, err := repository.FindByID(ctx, created.ID); !errors.Is(err, domain.ErrPrinterNotFound) {
		t.Fatalf("FindByID(deleted)=%v", err)
	}
	values.Name = "A1 Maintenance"
	maintainedPrinter, err := repository.Create(ctx, values, now)
	if err != nil {
		t.Fatalf("create maintained printer=%v", err)
	}
	var userID string
	if err := database.QueryRowContext(ctx, "INSERT INTO users(name,email_or_username,password_hash,status,role) VALUES('Maintenance Owner',$1,'$argon2id$test','active','owner') RETURNING id", "maintenance-owner-"+maintainedPrinter.ID).Scan(&userID); err != nil {
		t.Fatalf("create maintenance user=%v", err)
	}
	hours, cost := "1250.5", int64(4500)
	maintenanceRepository := NewMaintenanceRepository(database)
	event, err := maintenanceRepository.Create(ctx, maintainedPrinter.ID, userID, domainmaintenance.Values{Type: domainmaintenance.TypePreventive, PerformedAt: now, PrinterHours: &hours, Description: "Lubricate axes", CostCents: &cost, DowntimeMinutes: 30}, now)
	if err != nil || event.PrinterHours == nil || *event.PrinterHours != hours || event.CostCents == nil || *event.CostCents != cost || event.CreatedBy != userID {
		t.Fatalf("Create maintenance=%#v,%v", event, err)
	}
	events, err := maintenanceRepository.List(ctx, maintainedPrinter.ID)
	if err != nil || len(events) != 1 || events[0].Type != domainmaintenance.TypePreventive {
		t.Fatalf("List maintenance=%#v,%v", events, err)
	}
}
