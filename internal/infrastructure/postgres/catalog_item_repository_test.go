package postgres

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
)

func TestCatalogItemWhereUsesBoundParameters(t *testing.T) {
	purpose := domaincatalog.PurposeProduct
	status := domaincatalog.StatusActive
	sellable := true
	where, args := catalogItemWhere(domaincatalog.ListFilter{
		Purpose: &purpose, Status: &status, Sellable: &sellable,
		Tag: "tag' OR TRUE --", Query: "cube%_",
	})
	wantWhere := " WHERE purpose = $1 AND status = $2 AND sellable = $3 AND tags ? $4 AND " +
		"(position(lower($5) in lower(name)) > 0 OR position(lower($5) in lower(COALESCE(sku, ''))) > 0 OR position(lower($5) in lower(description)) > 0)"
	if where != wantWhere {
		t.Fatalf("where = %q, want %q", where, wantWhere)
	}
	if len(args) != 5 || args[3] != "tag' OR TRUE --" || args[4] != "cube%_" {
		t.Fatalf("args = %#v", args)
	}
}

func TestCatalogItemRepositoryAgainstPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("TALOS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TALOS_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	database, err := Open(ctx, testConfig(databaseURL))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = database.ExecContext(ctx, "TRUNCATE TABLE energy_measurements, print_job_material_usage, job_events, print_jobs, catalog_bom_items, design_version_files, design_versions, catalog_parts, catalog_items")
		_ = database.Close()
	})
	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE energy_measurements, print_job_material_usage, job_events, print_jobs, catalog_bom_items, design_version_files, design_versions, catalog_parts, catalog_items"); err != nil {
		t.Fatalf("truncate catalog_items: %v", err)
	}

	repository := NewCatalogItemRepository(database)
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	sku := "CUBE-01"
	created, err := repository.Create(ctx, domaincatalog.Values{
		Name: "Calibration Cube", SKU: &sku, Description: "Printer test",
		Purpose: domaincatalog.PurposeTest, Tags: []string{"calibration", "pla"}, Status: domaincatalog.StatusActive,
	}, now)
	if err != nil || created.ID == "" || created.SKU == nil || *created.SKU != sku || !created.CreatedAt.Equal(now) {
		t.Fatalf("Create() = %#v, %v", created, err)
	}
	found, err := repository.FindByID(ctx, created.ID)
	if err != nil || found.ID != created.ID || !reflect.DeepEqual(found.Tags, []string{"calibration", "pla"}) {
		t.Fatalf("FindByID() = %#v, %v", found, err)
	}

	purpose := domaincatalog.PurposeTest
	status := domaincatalog.StatusActive
	sellable := false
	page, err := repository.List(ctx, domaincatalog.ListFilter{
		Purpose: &purpose, Status: &status, Sellable: &sellable, Tag: "pla", Query: "cube", Limit: 10,
	})
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != created.ID {
		t.Fatalf("List() = %#v, %v", page, err)
	}

	updatedAt := now.Add(time.Hour)
	updated, err := repository.Update(ctx, created.ID, domaincatalog.Values{
		Name: "Archived Cube", Description: "Printer test", Purpose: domaincatalog.PurposeTest,
		Tags: []string{"calibration"}, Status: domaincatalog.StatusArchived,
	}, updatedAt)
	if err != nil || updated.Name != "Archived Cube" || updated.Status != domaincatalog.StatusArchived || !updated.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("Update() = %#v, %v", updated, err)
	}
	if err := repository.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repository.FindByID(ctx, created.ID); !errors.Is(err, domaincatalog.ErrItemNotFound) {
		t.Fatalf("FindByID() after delete error = %v", err)
	}
	if err := repository.Delete(ctx, created.ID); !errors.Is(err, domaincatalog.ErrItemNotFound) {
		t.Fatalf("Delete() missing error = %v", err)
	}
}
