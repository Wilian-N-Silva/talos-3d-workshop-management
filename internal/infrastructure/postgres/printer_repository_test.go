package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

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
		_, _ = database.ExecContext(ctx, "TRUNCATE TABLE job_events, print_jobs, printers")
		_ = database.Close()
	})
	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate()=%v", err)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE job_events, print_jobs, printers"); err != nil {
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
}
