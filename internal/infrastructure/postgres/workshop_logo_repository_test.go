package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
)

func TestWorkshopLogoRepositoryAgainstPostgreSQL(t *testing.T) {
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
		_, _ = database.ExecContext(ctx, "TRUNCATE TABLE workshop_settings, files, sessions, bootstrap_state, users, client_devices")
		if err := database.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE workshop_settings, files, sessions, bootstrap_state, users, client_devices"); err != nil {
		t.Fatalf("truncate logo tables: %v", err)
	}

	user, err := NewUserRepository(database).Create(ctx, domainauth.CreateUserParams{
		Name:            "Workshop Owner",
		EmailOrUsername: "owner@example.com",
		PasswordHash:    "$argon2id$test",
		Status:          domainauth.UserStatusActive,
		Role:            domainauth.RoleOwner,
	})
	if err != nil {
		t.Fatalf("create uploader: %v", err)
	}
	settingsRepository := NewWorkshopSettingsRepository(database)
	initialized, err := settingsRepository.Initialize(ctx, domainsettings.Values{
		WorkshopName:    "Prototype Lab",
		DefaultLocale:   "pt-BR",
		DefaultCurrency: "BRL",
		DisplayTimezone: "America/Sao_Paulo",
		DefaultTheme:    domainsettings.ThemeSystem,
	})
	if err != nil {
		t.Fatalf("initialize settings: %v", err)
	}

	repository := NewWorkshopLogoRepository(database)
	if _, err := repository.CurrentLogo(ctx); !errors.Is(err, domainfiles.ErrFileNotFound) {
		t.Fatalf("CurrentLogo() before association error = %v", err)
	}
	firstParams := logoFileParams(user.ID, "first-logo")
	first, settings, err := repository.AssociateLogo(ctx, firstParams, initialized.CreatedAt.Add(time.Minute))
	if err != nil {
		t.Fatalf("AssociateLogo() first error = %v", err)
	}
	if first.ID == "" || settings.LogoFileID == nil || *settings.LogoFileID != first.ID ||
		first.OriginalName != firstParams.OriginalName || first.UploadedBy != user.ID {
		t.Fatalf("first association = %#v, %#v", first, settings)
	}
	current, err := repository.CurrentLogo(ctx)
	if err != nil || current.ID != first.ID {
		t.Fatalf("CurrentLogo() = %#v, %v", current, err)
	}

	secondParams := logoFileParams(user.ID, "second-logo")
	second, settings, err := repository.AssociateLogo(ctx, secondParams, settings.UpdatedAt.Add(time.Minute))
	if err != nil || second.ID == first.ID || settings.LogoFileID == nil || *settings.LogoFileID != second.ID {
		t.Fatalf("second association = %#v, %#v, %v", second, settings, err)
	}
	var records int
	if err := database.QueryRowContext(ctx, "SELECT count(*) FROM files").Scan(&records); err != nil || records != 2 {
		t.Fatalf("file count after replacement = %d, %v", records, err)
	}
	var previousExists bool
	if err := database.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM files WHERE id = $1)", first.ID).Scan(&previousExists); err != nil || !previousExists {
		t.Fatalf("previous logo exists = %t, %v", previousExists, err)
	}

	deduplicated, settings, err := repository.AssociateLogo(ctx, firstParams, settings.UpdatedAt.Add(time.Minute))
	if err != nil || deduplicated.ID != first.ID || settings.LogoFileID == nil || *settings.LogoFileID != first.ID {
		t.Fatalf("deduplicated association = %#v, %#v, %v", deduplicated, settings, err)
	}
	if err := database.QueryRowContext(ctx, "SELECT count(*) FROM files").Scan(&records); err != nil || records != 2 {
		t.Fatalf("file count after deduplication = %d, %v", records, err)
	}

	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE workshop_settings"); err != nil {
		t.Fatalf("remove settings singleton: %v", err)
	}
	thirdParams := logoFileParams(user.ID, "third-logo")
	if _, _, err := repository.AssociateLogo(ctx, thirdParams, time.Now()); !errors.Is(err, domainsettings.ErrWorkshopSettingsNotFound) {
		t.Fatalf("AssociateLogo() without settings error = %v", err)
	}
	if err := database.QueryRowContext(ctx, "SELECT count(*) FROM files").Scan(&records); err != nil || records != 2 {
		t.Fatalf("file count after rolled-back association = %d, %v", records, err)
	}
}

func logoFileParams(uploaderID, content string) domainfiles.CreateParams {
	digest := sha256.Sum256([]byte(content))
	return domainfiles.CreateParams{
		SHA256:       digest[:],
		OriginalName: content + ".png",
		ContentType:  "image/png",
		SizeBytes:    int64(len(content)),
		StorageKey:   hex.EncodeToString(digest[:]),
		UploadedBy:   uploaderID,
	}
}
