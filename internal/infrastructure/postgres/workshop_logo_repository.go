package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
)

const fileColumns = `id, sha256, original_name, content_type, size_bytes, storage_key, uploaded_by, created_at`

// WorkshopLogoRepository atomically persists file metadata and workshop association.
type WorkshopLogoRepository struct {
	database *sql.DB
}

// NewWorkshopLogoRepository creates a PostgreSQL workshop-logo repository.
func NewWorkshopLogoRepository(database *sql.DB) *WorkshopLogoRepository {
	return &WorkshopLogoRepository{database: database}
}

// AssociateLogo creates or reuses immutable content metadata and selects it as
// the current logo in one database transaction.
func (repository *WorkshopLogoRepository) AssociateLogo(
	ctx context.Context,
	params domainfiles.CreateParams,
	updatedAt time.Time,
) (file domainfiles.File, settings domainsettings.WorkshopSettings, returnErr error) {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return domainfiles.File{}, domainsettings.WorkshopSettings{}, fmt.Errorf("begin workshop logo transaction: %w", err)
	}
	defer func() {
		if err := transaction.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			returnErr = errors.Join(returnErr, fmt.Errorf("rollback workshop logo transaction: %w", err))
		}
	}()

	file, err = scanFile(transaction.QueryRowContext(
		ctx,
		`INSERT INTO files (sha256, original_name, content_type, size_bytes, storage_key, uploaded_by)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (sha256) DO NOTHING
		 RETURNING `+fileColumns,
		params.SHA256,
		params.OriginalName,
		params.ContentType,
		params.SizeBytes,
		params.StorageKey,
		params.UploadedBy,
	))
	if errors.Is(err, sql.ErrNoRows) {
		file, err = scanFile(transaction.QueryRowContext(
			ctx,
			`SELECT `+fileColumns+` FROM files WHERE sha256 = $1`,
			params.SHA256,
		))
	}
	if err != nil {
		return domainfiles.File{}, domainsettings.WorkshopSettings{}, fmt.Errorf("persist workshop logo metadata: %w", err)
	}

	settings, err = scanWorkshopSettings(transaction.QueryRowContext(
		ctx,
		`UPDATE workshop_settings
		 SET logo_file_id = $1,
		     updated_at = GREATEST(updated_at, $2)
		 WHERE singleton_id = 1
		 RETURNING `+workshopSettingsColumns,
		file.ID,
		updatedAt.UTC(),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domainfiles.File{}, domainsettings.WorkshopSettings{}, domainsettings.ErrWorkshopSettingsNotFound
	}
	if err != nil {
		return domainfiles.File{}, domainsettings.WorkshopSettings{}, fmt.Errorf("associate workshop logo metadata: %w", err)
	}
	if err := transaction.Commit(); err != nil {
		return domainfiles.File{}, domainsettings.WorkshopSettings{}, fmt.Errorf("commit workshop logo transaction: %w", err)
	}
	return file, settings, nil
}

// CurrentLogo returns only the immutable file associated with current branding.
func (repository *WorkshopLogoRepository) CurrentLogo(ctx context.Context) (domainfiles.File, error) {
	file, err := scanFile(repository.database.QueryRowContext(
		ctx,
		`SELECT f.id, f.sha256, f.original_name, f.content_type, f.size_bytes,
		        f.storage_key, f.uploaded_by, f.created_at
		 FROM workshop_settings AS s
		 JOIN files AS f ON f.id = s.logo_file_id
		 WHERE s.singleton_id = 1`,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domainfiles.File{}, domainfiles.ErrFileNotFound
	}
	if err != nil {
		return domainfiles.File{}, fmt.Errorf("get current workshop logo: %w", err)
	}
	return file, nil
}

func scanFile(row rowScanner) (domainfiles.File, error) {
	var file domainfiles.File
	if err := row.Scan(
		&file.ID,
		&file.SHA256,
		&file.OriginalName,
		&file.ContentType,
		&file.SizeBytes,
		&file.StorageKey,
		&file.UploadedBy,
		&file.CreatedAt,
	); err != nil {
		return domainfiles.File{}, err
	}
	file.SHA256 = append([]byte(nil), file.SHA256...)
	file.CreatedAt = file.CreatedAt.UTC()
	return file, nil
}
