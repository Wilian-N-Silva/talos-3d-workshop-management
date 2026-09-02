package postgres

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
)

func TestWorkshopSettingsRepositoryAgainstPostgreSQL(t *testing.T) {
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
		if err := database.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	if err := Migrate(ctx, database); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE workshop_settings"); err != nil {
		t.Fatalf("truncate workshop settings: %v", err)
	}

	repository := NewWorkshopSettingsRepository(database)
	if _, err := repository.Get(ctx); !errors.Is(err, domainsettings.ErrWorkshopSettingsNotFound) {
		t.Fatalf("Get() before initialize error = %v", err)
	}

	var defaultSettings domainsettings.WorkshopSettings
	var defaultLogoFileID sql.NullString
	if err := database.QueryRowContext(
		ctx,
		`INSERT INTO workshop_settings DEFAULT VALUES
		 RETURNING `+workshopSettingsColumns,
	).Scan(
		&defaultSettings.WorkshopName,
		&defaultLogoFileID,
		&defaultSettings.DefaultLocale,
		&defaultSettings.DefaultCurrency,
		&defaultSettings.DisplayTimezone,
		&defaultSettings.DefaultTheme,
		&defaultSettings.CreatedAt,
		&defaultSettings.UpdatedAt,
	); err != nil {
		t.Fatalf("insert schema defaults: %v", err)
	}
	if defaultSettings.WorkshopName != "Workshop" || defaultSettings.DefaultLocale != "pt-BR" ||
		defaultSettings.DefaultCurrency != "BRL" || defaultSettings.DisplayTimezone != "America/Sao_Paulo" ||
		defaultSettings.DefaultTheme != domainsettings.ThemeSystem {
		t.Fatalf("schema defaults = %#v", defaultSettings)
	}
	if _, err := database.ExecContext(ctx, "TRUNCATE TABLE workshop_settings"); err != nil {
		t.Fatalf("reset schema defaults: %v", err)
	}

	configured := domainsettings.Values{
		WorkshopName:    "Prototype Lab",
		DefaultLocale:   "en-US",
		DefaultCurrency: "USD",
		DisplayTimezone: "UTC",
		DefaultTheme:    domainsettings.ThemeDark,
	}
	initialized, err := repository.Initialize(ctx, configured)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if initialized.WorkshopName != configured.WorkshopName || initialized.DefaultTheme != configured.DefaultTheme || initialized.LogoFileID != nil {
		t.Fatalf("Initialize() = %#v", initialized)
	}

	secondDefaults := domainsettings.Values{
		WorkshopName:    "Must Not Replace",
		DefaultLocale:   "pt-BR",
		DefaultCurrency: "BRL",
		DisplayTimezone: "America/Sao_Paulo",
		DefaultTheme:    domainsettings.ThemeSystem,
	}
	preserved, err := repository.Initialize(ctx, secondDefaults)
	if err != nil || preserved.WorkshopName != configured.WorkshopName {
		t.Fatalf("Initialize() repeated = %#v, %v", preserved, err)
	}
	var records int
	if err := database.QueryRowContext(ctx, "SELECT count(*) FROM workshop_settings").Scan(&records); err != nil || records != 1 {
		t.Fatalf("settings count = %d, %v", records, err)
	}

	updatedAt := initialized.CreatedAt.Add(time.Minute)
	updated, err := repository.Update(ctx, secondDefaults, updatedAt)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.WorkshopName != secondDefaults.WorkshopName || !updated.UpdatedAt.Equal(updatedAt) || updated.LogoFileID != nil {
		t.Fatalf("Update() = %#v", updated)
	}

	if _, err := database.ExecContext(
		ctx,
		`INSERT INTO workshop_settings (singleton_id) VALUES (2)`,
	); err == nil {
		t.Fatal("insert second logical settings record error = nil")
	}
	if _, err := database.ExecContext(
		ctx,
		`UPDATE workshop_settings SET default_theme = 'custom' WHERE singleton_id = 1`,
	); err == nil {
		t.Fatal("invalid theme update error = nil")
	}
}
