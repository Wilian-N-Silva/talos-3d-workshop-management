package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
)

// FileRepository persists and locates immutable file metadata.
type FileRepository struct {
	database *sql.DB
}

// NewFileRepository creates a PostgreSQL immutable-file repository.
func NewFileRepository(database *sql.DB) *FileRepository {
	return &FileRepository{database: database}
}

// CreateOrGet inserts metadata or returns the existing record for the same
// SHA-256 digest. The first successful upload owns immutable descriptive metadata.
func (repository *FileRepository) CreateOrGet(
	ctx context.Context,
	params domainfiles.CreateParams,
) (domainfiles.File, bool, error) {
	file, err := scanFile(repository.database.QueryRowContext(
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
	if err == nil {
		return file, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domainfiles.File{}, false, fmt.Errorf("create file metadata: %w", err)
	}

	file, err = scanFile(repository.database.QueryRowContext(
		ctx,
		`SELECT `+fileColumns+` FROM files WHERE sha256 = $1`,
		params.SHA256,
	))
	if err != nil {
		return domainfiles.File{}, false, fmt.Errorf("get deduplicated file metadata: %w", err)
	}
	return file, false, nil
}

// FindByID returns immutable metadata for one file UUID.
func (repository *FileRepository) FindByID(ctx context.Context, id string) (domainfiles.File, error) {
	file, err := scanFile(repository.database.QueryRowContext(
		ctx,
		`SELECT `+fileColumns+` FROM files WHERE id = $1`,
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domainfiles.File{}, domainfiles.ErrFileNotFound
	}
	if err != nil {
		return domainfiles.File{}, fmt.Errorf("find file metadata: %w", err)
	}
	return file, nil
}
