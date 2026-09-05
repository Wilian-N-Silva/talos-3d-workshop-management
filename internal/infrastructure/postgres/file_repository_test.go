package postgres

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
)

func TestFileRepositoryAgainstPostgreSQL(t *testing.T) {
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
		_, _ = database.ExecContext(ctx, "TRUNCATE TABLE job_labor_entries, labor_rates, energy_measurements, print_job_material_usage, job_events, print_jobs, catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, workshop_settings, files, sessions, bootstrap_state, users, client_devices")
		_ = database.Close()
	})
	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE job_labor_entries, labor_rates, energy_measurements, print_job_material_usage, job_events, print_jobs, catalog_bom_items, supply_movements, supplies, spool_measurements, material_spools, materials, design_version_files, design_versions, catalog_parts, catalog_items, workshop_settings, files, sessions, bootstrap_state, users, client_devices"); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
	user, err := NewUserRepository(database).Create(ctx, domainauth.CreateUserParams{
		Name: "Designer", EmailOrUsername: "designer", PasswordHash: "$argon2id$test",
		Status: domainauth.UserStatusActive, Role: domainauth.RoleDesigner,
	})
	if err != nil {
		t.Fatalf("create uploader: %v", err)
	}
	digest := sha256.Sum256([]byte("mesh content"))
	params := domainfiles.CreateParams{SHA256: digest[:], OriginalName: "part.stl", ContentType: "application/octet-stream", SizeBytes: 12, StorageKey: "meshcontentkey", UploadedBy: user.ID}
	repository := NewFileRepository(database)

	created, wasCreated, err := repository.CreateOrGet(ctx, params)
	if err != nil || !wasCreated || created.ID == "" || created.OriginalName != "part.stl" || created.UploadedBy != user.ID {
		t.Fatalf("CreateOrGet() = %#v, %t, %v", created, wasCreated, err)
	}
	deduplicated, wasCreated, err := repository.CreateOrGet(ctx, domainfiles.CreateParams{
		SHA256: digest[:], OriginalName: "renamed.stl", ContentType: "model/stl", SizeBytes: 12, StorageKey: "differentkey", UploadedBy: user.ID,
	})
	if err != nil || wasCreated || deduplicated.ID != created.ID || deduplicated.OriginalName != "part.stl" {
		t.Fatalf("CreateOrGet() duplicate = %#v, %t, %v", deduplicated, wasCreated, err)
	}
	found, err := repository.FindByID(ctx, created.ID)
	if err != nil || found.ID != created.ID || found.StorageKey != params.StorageKey {
		t.Fatalf("FindByID() = %#v, %v", found, err)
	}
	if _, err := repository.FindByID(ctx, "11111111-1111-4111-8111-111111111111"); !errors.Is(err, domainfiles.ErrFileNotFound) {
		t.Fatalf("FindByID() missing error = %v", err)
	}
}
