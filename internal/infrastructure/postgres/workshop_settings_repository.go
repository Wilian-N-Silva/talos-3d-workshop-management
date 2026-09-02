package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
)

const workshopSettingsColumns = `workshop_name, logo_file_id, default_locale, default_currency,
display_timezone, default_theme, created_at, updated_at`

// WorkshopSettingsRepository persists the singleton workshop settings record.
type WorkshopSettingsRepository struct {
	database *sql.DB
}

// NewWorkshopSettingsRepository creates a PostgreSQL settings repository.
func NewWorkshopSettingsRepository(database *sql.DB) *WorkshopSettingsRepository {
	return &WorkshopSettingsRepository{database: database}
}

// Initialize inserts configured defaults once and returns the durable record.
func (repository *WorkshopSettingsRepository) Initialize(
	ctx context.Context,
	values domainsettings.Values,
) (domainsettings.WorkshopSettings, error) {
	if _, err := repository.database.ExecContext(
		ctx,
		`INSERT INTO workshop_settings (
		     singleton_id, workshop_name, default_locale, default_currency,
		     display_timezone, default_theme
		 ) VALUES (1, $1, $2, $3, $4, $5)
		 ON CONFLICT (singleton_id) DO NOTHING`,
		values.WorkshopName,
		values.DefaultLocale,
		values.DefaultCurrency,
		values.DisplayTimezone,
		values.DefaultTheme,
	); err != nil {
		return domainsettings.WorkshopSettings{}, fmt.Errorf("insert workshop settings defaults: %w", err)
	}
	return repository.Get(ctx)
}

// Get returns the singleton workshop settings record.
func (repository *WorkshopSettingsRepository) Get(
	ctx context.Context,
) (domainsettings.WorkshopSettings, error) {
	settings, err := scanWorkshopSettings(repository.database.QueryRowContext(
		ctx,
		`SELECT `+workshopSettingsColumns+` FROM workshop_settings WHERE singleton_id = 1`,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domainsettings.WorkshopSettings{}, domainsettings.ErrWorkshopSettingsNotFound
	}
	if err != nil {
		return domainsettings.WorkshopSettings{}, fmt.Errorf("get workshop settings: %w", err)
	}
	return settings, nil
}

// Update replaces mutable presentation settings without changing logo association.
func (repository *WorkshopSettingsRepository) Update(
	ctx context.Context,
	values domainsettings.Values,
	updatedAt time.Time,
) (domainsettings.WorkshopSettings, error) {
	settings, err := scanWorkshopSettings(repository.database.QueryRowContext(
		ctx,
		`UPDATE workshop_settings
		 SET workshop_name = $1,
		     default_locale = $2,
		     default_currency = $3,
		     display_timezone = $4,
		     default_theme = $5,
		     updated_at = GREATEST(updated_at, $6)
		 WHERE singleton_id = 1
		 RETURNING `+workshopSettingsColumns,
		values.WorkshopName,
		values.DefaultLocale,
		values.DefaultCurrency,
		values.DisplayTimezone,
		values.DefaultTheme,
		updatedAt.UTC(),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domainsettings.WorkshopSettings{}, domainsettings.ErrWorkshopSettingsNotFound
	}
	if err != nil {
		return domainsettings.WorkshopSettings{}, fmt.Errorf("update workshop settings: %w", err)
	}
	return settings, nil
}

func scanWorkshopSettings(row rowScanner) (domainsettings.WorkshopSettings, error) {
	var settings domainsettings.WorkshopSettings
	var logoFileID sql.NullString
	if err := row.Scan(
		&settings.WorkshopName,
		&logoFileID,
		&settings.DefaultLocale,
		&settings.DefaultCurrency,
		&settings.DisplayTimezone,
		&settings.DefaultTheme,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	); err != nil {
		return domainsettings.WorkshopSettings{}, err
	}
	if logoFileID.Valid {
		settings.LogoFileID = &logoFileID.String
	}
	settings.CreatedAt = settings.CreatedAt.UTC()
	settings.UpdatedAt = settings.UpdatedAt.UTC()
	return settings, nil
}
