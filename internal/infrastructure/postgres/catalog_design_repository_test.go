package postgres

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
)

func TestCatalogDesignRepositoryAgainstPostgreSQL(t *testing.T) {
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
		_, _ = database.ExecContext(ctx, "TRUNCATE TABLE maintenance_events, job_labor_entries, labor_rates, energy_measurements, print_job_material_usage, job_events, print_jobs, catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, workshop_settings, files, sessions, bootstrap_state, users, client_devices")
		_ = database.Close()
	})
	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE maintenance_events, job_labor_entries, labor_rates, energy_measurements, print_job_material_usage, job_events, print_jobs, catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, workshop_settings, files, sessions, bootstrap_state, users, client_devices"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	user, err := NewUserRepository(database).Create(ctx, domainauth.CreateUserParams{Name: "Designer", EmailOrUsername: "designer-design", PasswordHash: "$argon2id$test", Status: domainauth.UserStatusActive, Role: domainauth.RoleDesigner})
	if err != nil {
		t.Fatalf("create designer: %v", err)
	}
	item, err := NewCatalogItemRepository(database).Create(ctx, domaincatalog.Values{Name: "Tag", Purpose: domaincatalog.PurposeProduct, Sellable: true, Tags: []string{}, Status: domaincatalog.StatusActive}, time.Now().UTC())
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	digest := sha256.Sum256([]byte("print file"))
	file, _, err := NewFileRepository(database).CreateOrGet(ctx, domainfiles.CreateParams{SHA256: digest[:], OriginalName: "tag.3mf", ContentType: "application/octet-stream", SizeBytes: 10, StorageKey: "catalogdesignfilekey", UploadedBy: user.ID})
	if err != nil {
		t.Fatalf("create file: %v", err)
	}

	repository := NewCatalogDesignRepository(database)
	now := time.Date(2026, 9, 4, 16, 0, 0, 0, time.UTC)
	part, err := repository.CreatePart(ctx, item.ID, domaincatalog.PartValues{Name: "Body", Quantity: 2, Notes: "pair"}, now)
	if err != nil || part.ID == "" || part.CatalogItemID != item.ID || part.Quantity != 2 {
		t.Fatalf("CreatePart() = %#v, %v", part, err)
	}
	foundPart, err := repository.FindPart(ctx, part.ID)
	if err != nil || foundPart.Name != "Body" {
		t.Fatalf("FindPart() = %#v, %v", foundPart, err)
	}
	parts, err := repository.ListParts(ctx, item.ID)
	if err != nil || len(parts) != 1 || parts[0].ID != part.ID {
		t.Fatalf("ListParts() = %#v, %v", parts, err)
	}
	updated, err := repository.UpdatePart(ctx, part.ID, domaincatalog.PartValues{Name: "Shell", Quantity: 1}, now.Add(time.Hour))
	if err != nil || updated.Name != "Shell" || updated.Quantity != 1 {
		t.Fatalf("UpdatePart() = %#v, %v", updated, err)
	}

	commercial := false
	versionValues := domaincatalog.DesignVersionValues{Version: "v1", Origin: domaincatalog.DesignOriginThirdParty, LicenseName: "CC BY-NC", CommercialUseAllowed: &commercial}
	version, err := repository.CreateDesignVersion(ctx, part.ID, user.ID, versionValues, now)
	if err != nil || version.ID == "" || version.CreatedBy != user.ID || version.CommercialUseAllowed == nil || *version.CommercialUseAllowed {
		t.Fatalf("CreateDesignVersion() = %#v, %v", version, err)
	}
	if _, err := repository.CreateDesignVersion(ctx, part.ID, user.ID, versionValues, now); !errors.Is(err, domaincatalog.ErrDesignVersionConflict) {
		t.Fatalf("duplicate version error = %v", err)
	}
	linked, err := repository.AttachDesignFile(ctx, version.ID, file.ID, user.ID, domaincatalog.DesignFilePrint, now)
	if err != nil || linked.FileID != file.ID || linked.Role != domaincatalog.DesignFilePrint || linked.SHA256Hex == "" {
		t.Fatalf("AttachDesignFile() = %#v, %v", linked, err)
	}
	if _, err := repository.AttachDesignFile(ctx, version.ID, file.ID, user.ID, domaincatalog.DesignFilePrint, now); !errors.Is(err, domaincatalog.ErrDesignFileConflict) {
		t.Fatalf("duplicate link error = %v", err)
	}
	version2, err := repository.CreateDesignVersion(ctx, part.ID, user.ID, domaincatalog.DesignVersionValues{Version: "v2", Origin: domaincatalog.DesignOriginOriginal}, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("create v2: %v", err)
	}
	if _, err := repository.AttachDesignFile(ctx, version2.ID, file.ID, user.ID, domaincatalog.DesignFileSource, now); err != nil {
		t.Fatalf("reuse file: %v", err)
	}
	history, err := repository.ListDesignVersions(ctx, part.ID)
	if err != nil || len(history) != 2 || len(history[1].Files) != 1 || history[1].Files[0].Role != domaincatalog.DesignFilePrint {
		t.Fatalf("ListDesignVersions() = %#v, %v", history, err)
	}

	if err := repository.DeletePart(ctx, part.ID); !errors.Is(err, domaincatalog.ErrDesignHistoryExists) {
		t.Fatalf("DeletePart() with history error = %v", err)
	}
	if err := NewCatalogItemRepository(database).Delete(ctx, item.ID); !errors.Is(err, domaincatalog.ErrDesignHistoryExists) {
		t.Fatalf("Delete() item with history error = %v", err)
	}
	emptyItem, err := NewCatalogItemRepository(database).Create(ctx, domaincatalog.Values{Name: "Empty", Purpose: domaincatalog.PurposePrototype, Tags: []string{}, Status: domaincatalog.StatusActive}, now)
	if err != nil {
		t.Fatalf("create empty item: %v", err)
	}
	emptyPart, err := repository.CreatePart(ctx, emptyItem.ID, domaincatalog.PartValues{Name: "Draft", Quantity: 1}, now)
	if err != nil {
		t.Fatalf("create empty part: %v", err)
	}
	if err := NewCatalogItemRepository(database).Delete(ctx, emptyItem.ID); err != nil {
		t.Fatalf("delete empty item cascade: %v", err)
	}
	if _, err := repository.FindPart(ctx, emptyPart.ID); !errors.Is(err, domaincatalog.ErrPartNotFound) {
		t.Fatalf("cascaded part error = %v", err)
	}

	var partCount, versionCount, linkCount int
	_ = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM catalog_parts").Scan(&partCount)
	_ = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM design_versions").Scan(&versionCount)
	_ = database.QueryRowContext(ctx, "SELECT COUNT(*) FROM design_version_files").Scan(&linkCount)
	if partCount != 1 || versionCount != 2 || linkCount != 2 {
		t.Fatalf("preserved history counts = parts %d, versions %d, links %d", partCount, versionCount, linkCount)
	}
}
