package postgres

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
	"github.com/jackc/pgx/v5/pgconn"
)

const partColumns = `id, catalog_item_id, name, quantity, notes, created_at, updated_at`
const versionColumns = `id, catalog_part_id, version, notes, origin, source_url, original_author, license_name, commercial_use_allowed, attribution_required, attribution_text, created_by, created_at`

type CatalogDesignRepository struct {
	database *sql.DB
}

func NewCatalogDesignRepository(database *sql.DB) *CatalogDesignRepository {
	return &CatalogDesignRepository{database: database}
}

func (repository *CatalogDesignRepository) CreatePart(ctx context.Context, itemID string, values domaincatalog.PartValues, now time.Time) (domaincatalog.Part, error) {
	part, err := scanPart(repository.database.QueryRowContext(ctx,
		`INSERT INTO catalog_parts (catalog_item_id, name, quantity, notes, created_at, updated_at)
		 SELECT id, $2, $3, $4, $5, $5 FROM catalog_items WHERE id = $1
		 RETURNING `+partColumns,
		itemID, values.Name, values.Quantity, values.Notes, now.UTC(),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domaincatalog.Part{}, domaincatalog.ErrItemNotFound
	}
	if err != nil {
		return domaincatalog.Part{}, fmt.Errorf("insert catalog part: %w", err)
	}
	return part, nil
}

func (repository *CatalogDesignRepository) FindPart(ctx context.Context, partID string) (domaincatalog.Part, error) {
	part, err := scanPart(repository.database.QueryRowContext(ctx, `SELECT `+partColumns+` FROM catalog_parts WHERE id = $1`, partID))
	if errors.Is(err, sql.ErrNoRows) {
		return domaincatalog.Part{}, domaincatalog.ErrPartNotFound
	}
	if err != nil {
		return domaincatalog.Part{}, fmt.Errorf("find catalog part: %w", err)
	}
	return part, nil
}

func (repository *CatalogDesignRepository) ListParts(ctx context.Context, itemID string) ([]domaincatalog.Part, error) {
	exists, err := repository.exists(ctx, "catalog_items", itemID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domaincatalog.ErrItemNotFound
	}
	rows, err := repository.database.QueryContext(ctx, `SELECT `+partColumns+` FROM catalog_parts WHERE catalog_item_id = $1 ORDER BY name, id`, itemID)
	if err != nil {
		return nil, fmt.Errorf("query catalog parts: %w", err)
	}
	defer rows.Close()
	parts := []domaincatalog.Part{}
	for rows.Next() {
		part, err := scanPart(rows)
		if err != nil {
			return nil, fmt.Errorf("scan catalog part: %w", err)
		}
		parts = append(parts, part)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate catalog parts: %w", err)
	}
	return parts, nil
}

func (repository *CatalogDesignRepository) UpdatePart(ctx context.Context, partID string, values domaincatalog.PartValues, now time.Time) (domaincatalog.Part, error) {
	part, err := scanPart(repository.database.QueryRowContext(ctx,
		`UPDATE catalog_parts SET name = $2, quantity = $3, notes = $4, updated_at = GREATEST(updated_at, $5) WHERE id = $1 RETURNING `+partColumns,
		partID, values.Name, values.Quantity, values.Notes, now.UTC(),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domaincatalog.Part{}, domaincatalog.ErrPartNotFound
	}
	if err != nil {
		return domaincatalog.Part{}, fmt.Errorf("update catalog part: %w", err)
	}
	return part, nil
}

func (repository *CatalogDesignRepository) DeletePart(ctx context.Context, partID string) error {
	result, err := repository.database.ExecContext(ctx, "DELETE FROM catalog_parts WHERE id = $1", partID)
	if foreignKeyViolation(err, "design_versions_part_fk") {
		return domaincatalog.ErrDesignHistoryExists
	}
	if err != nil {
		return fmt.Errorf("delete catalog part: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted catalog part count: %w", err)
	}
	if count == 0 {
		return domaincatalog.ErrPartNotFound
	}
	return nil
}

func (repository *CatalogDesignRepository) CreateDesignVersion(ctx context.Context, partID, actorID string, values domaincatalog.DesignVersionValues, now time.Time) (domaincatalog.DesignVersion, error) {
	version, err := scanDesignVersion(repository.database.QueryRowContext(ctx,
		`INSERT INTO design_versions (catalog_part_id, version, notes, origin, source_url, original_author, license_name, commercial_use_allowed, attribution_required, attribution_text, created_by, created_at)
		 SELECT id, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12 FROM catalog_parts WHERE id = $1
		 RETURNING `+versionColumns,
		partID, values.Version, values.Notes, values.Origin, values.SourceURL, values.OriginalAuthor, values.LicenseName,
		values.CommercialUseAllowed, values.AttributionRequired, values.AttributionText, actorID, now.UTC(),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domaincatalog.DesignVersion{}, domaincatalog.ErrPartNotFound
	}
	if constraintViolation(err, "design_versions_part_version_unique") {
		return domaincatalog.DesignVersion{}, domaincatalog.ErrDesignVersionConflict
	}
	if err != nil {
		return domaincatalog.DesignVersion{}, fmt.Errorf("insert design version: %w", err)
	}
	version.Files = []domaincatalog.DesignFile{}
	return version, nil
}

func (repository *CatalogDesignRepository) ListDesignVersions(ctx context.Context, partID string) ([]domaincatalog.DesignVersion, error) {
	exists, err := repository.exists(ctx, "catalog_parts", partID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domaincatalog.ErrPartNotFound
	}
	rows, err := repository.database.QueryContext(ctx, `SELECT `+versionColumns+` FROM design_versions WHERE catalog_part_id = $1 ORDER BY created_at DESC, id DESC LIMIT 100`, partID)
	if err != nil {
		return nil, fmt.Errorf("query design versions: %w", err)
	}
	defer rows.Close()
	versions := []domaincatalog.DesignVersion{}
	for rows.Next() {
		version, err := scanDesignVersion(rows)
		if err != nil {
			return nil, fmt.Errorf("scan design version: %w", err)
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("iterate design versions: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close design versions: %w", err)
	}
	for index := range versions {
		versions[index].Files, err = repository.listDesignFiles(ctx, versions[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return versions, nil
}

func (repository *CatalogDesignRepository) AttachDesignFile(ctx context.Context, versionID, fileID, actorID string, role domaincatalog.DesignFileRole, now time.Time) (domaincatalog.DesignFile, error) {
	row := repository.database.QueryRowContext(ctx,
		`WITH attached AS (
		   INSERT INTO design_version_files (design_version_id, file_id, role, attached_by, created_at)
		   SELECT $1, $2, $3, $4, $5
		   WHERE EXISTS (SELECT 1 FROM design_versions WHERE id = $1)
		     AND EXISTS (SELECT 1 FROM files WHERE id = $2)
		   RETURNING file_id, role, created_at
		 )
		 SELECT attached.file_id, attached.role, files.original_name, files.content_type, files.size_bytes,
		        encode(files.sha256, 'hex'), attached.created_at
		 FROM attached JOIN files ON files.id = attached.file_id`,
		versionID, fileID, role, actorID, now.UTC(),
	)
	file, err := scanDesignFile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domaincatalog.DesignFile{}, domaincatalog.ErrDesignFileNotFound
	}
	if constraintViolation(err, "design_version_files_pkey") {
		return domaincatalog.DesignFile{}, domaincatalog.ErrDesignFileConflict
	}
	if err != nil {
		return domaincatalog.DesignFile{}, fmt.Errorf("insert design file link: %w", err)
	}
	return file, nil
}

func (repository *CatalogDesignRepository) listDesignFiles(ctx context.Context, versionID string) ([]domaincatalog.DesignFile, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT links.file_id, links.role, files.original_name, files.content_type, files.size_bytes,
		        encode(files.sha256, 'hex'), links.created_at
		 FROM design_version_files links JOIN files ON files.id = links.file_id
		 WHERE links.design_version_id = $1 ORDER BY links.role, links.created_at, links.file_id`, versionID)
	if err != nil {
		return nil, fmt.Errorf("query design files: %w", err)
	}
	defer rows.Close()
	files := []domaincatalog.DesignFile{}
	for rows.Next() {
		file, err := scanDesignFile(rows)
		if err != nil {
			return nil, fmt.Errorf("scan design file: %w", err)
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate design files: %w", err)
	}
	return files, nil
}

func (repository *CatalogDesignRepository) exists(ctx context.Context, table, id string) (bool, error) {
	var exists bool
	if err := repository.database.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM "+table+" WHERE id = $1)", id).Scan(&exists); err != nil {
		return false, fmt.Errorf("check %s existence: %w", table, err)
	}
	return exists, nil
}

func scanPart(row rowScanner) (domaincatalog.Part, error) {
	var part domaincatalog.Part
	if err := row.Scan(&part.ID, &part.CatalogItemID, &part.Name, &part.Quantity, &part.Notes, &part.CreatedAt, &part.UpdatedAt); err != nil {
		return domaincatalog.Part{}, err
	}
	part.CreatedAt, part.UpdatedAt = part.CreatedAt.UTC(), part.UpdatedAt.UTC()
	return part, nil
}

func scanDesignVersion(row rowScanner) (domaincatalog.DesignVersion, error) {
	var version domaincatalog.DesignVersion
	var sourceURL sql.NullString
	var commercialUse sql.NullBool
	if err := row.Scan(&version.ID, &version.CatalogPartID, &version.Version, &version.Notes, &version.Origin, &sourceURL,
		&version.OriginalAuthor, &version.LicenseName, &commercialUse, &version.AttributionRequired, &version.AttributionText,
		&version.CreatedBy, &version.CreatedAt); err != nil {
		return domaincatalog.DesignVersion{}, err
	}
	if sourceURL.Valid {
		version.SourceURL = &sourceURL.String
	}
	if commercialUse.Valid {
		value := commercialUse.Bool
		version.CommercialUseAllowed = &value
	}
	version.CreatedAt = version.CreatedAt.UTC()
	return version, nil
}

func scanDesignFile(row rowScanner) (domaincatalog.DesignFile, error) {
	var file domaincatalog.DesignFile
	if err := row.Scan(&file.FileID, &file.Role, &file.OriginalName, &file.ContentType, &file.SizeBytes, &file.SHA256Hex, &file.CreatedAt); err != nil {
		return domaincatalog.DesignFile{}, err
	}
	if _, err := hex.DecodeString(file.SHA256Hex); err != nil || len(file.SHA256Hex) != 64 {
		return domaincatalog.DesignFile{}, errors.New("invalid persisted file digest")
	}
	file.CreatedAt = file.CreatedAt.UTC()
	return file, nil
}

func constraintViolation(err error, constraint string) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && pgError.Code == "23505" && pgError.ConstraintName == constraint
}

func foreignKeyViolation(err error, constraint string) bool {
	var pgError *pgconn.PgError
	return errors.As(err, &pgError) && pgError.Code == "23503" && pgError.ConstraintName == constraint
}
