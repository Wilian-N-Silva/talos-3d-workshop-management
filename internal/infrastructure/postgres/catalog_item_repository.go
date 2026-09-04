package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
)

const catalogItemColumns = `id, name, sku, description, purpose, sellable, tags, status, created_at, updated_at`

// CatalogItemRepository persists generic catalog items in PostgreSQL.
type CatalogItemRepository struct {
	database *sql.DB
}

// NewCatalogItemRepository creates a PostgreSQL catalog item repository.
func NewCatalogItemRepository(database *sql.DB) *CatalogItemRepository {
	return &CatalogItemRepository{database: database}
}

// Create inserts one catalog item.
func (repository *CatalogItemRepository) Create(
	ctx context.Context,
	values domaincatalog.Values,
	now time.Time,
) (domaincatalog.Item, error) {
	tags, err := json.Marshal(values.Tags)
	if err != nil {
		return domaincatalog.Item{}, fmt.Errorf("encode catalog tags: %w", err)
	}
	item, err := scanCatalogItem(repository.database.QueryRowContext(
		ctx,
		`INSERT INTO catalog_items (name, sku, description, purpose, sellable, tags, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		 RETURNING `+catalogItemColumns,
		values.Name, values.SKU, values.Description, values.Purpose, values.Sellable, string(tags), values.Status, now.UTC(),
	))
	if err != nil {
		return domaincatalog.Item{}, fmt.Errorf("insert catalog item: %w", err)
	}
	return item, nil
}

// FindByID returns one catalog item by UUID.
func (repository *CatalogItemRepository) FindByID(ctx context.Context, id string) (domaincatalog.Item, error) {
	item, err := scanCatalogItem(repository.database.QueryRowContext(
		ctx,
		`SELECT `+catalogItemColumns+` FROM catalog_items WHERE id = $1`,
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domaincatalog.Item{}, domaincatalog.ErrItemNotFound
	}
	if err != nil {
		return domaincatalog.Item{}, fmt.Errorf("find catalog item: %w", err)
	}
	return item, nil
}

// List returns filtered catalog items and the total number of matching rows.
func (repository *CatalogItemRepository) List(
	ctx context.Context,
	filter domaincatalog.ListFilter,
) (domaincatalog.Page, error) {
	where, args := catalogItemWhere(filter)
	var total int64
	if err := repository.database.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM catalog_items"+where,
		args...,
	).Scan(&total); err != nil {
		return domaincatalog.Page{}, fmt.Errorf("count catalog items: %w", err)
	}

	limitPosition := len(args) + 1
	offsetPosition := len(args) + 2
	queryArgs := append(append([]any{}, args...), filter.Limit, filter.Offset)
	rows, err := repository.database.QueryContext(
		ctx,
		fmt.Sprintf(
			"SELECT %s FROM catalog_items%s ORDER BY name, id LIMIT $%d OFFSET $%d",
			catalogItemColumns, where, limitPosition, offsetPosition,
		),
		queryArgs...,
	)
	if err != nil {
		return domaincatalog.Page{}, fmt.Errorf("query catalog items: %w", err)
	}
	defer rows.Close()

	items := make([]domaincatalog.Item, 0, filter.Limit)
	for rows.Next() {
		item, err := scanCatalogItem(rows)
		if err != nil {
			return domaincatalog.Page{}, fmt.Errorf("scan catalog item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domaincatalog.Page{}, fmt.Errorf("iterate catalog items: %w", err)
	}
	return domaincatalog.Page{Items: items, Total: total, Limit: filter.Limit, Offset: filter.Offset}, nil
}

// Update replaces mutable fields and returns the durable item.
func (repository *CatalogItemRepository) Update(
	ctx context.Context,
	id string,
	values domaincatalog.Values,
	updatedAt time.Time,
) (domaincatalog.Item, error) {
	tags, err := json.Marshal(values.Tags)
	if err != nil {
		return domaincatalog.Item{}, fmt.Errorf("encode catalog tags: %w", err)
	}
	item, err := scanCatalogItem(repository.database.QueryRowContext(
		ctx,
		`UPDATE catalog_items
		 SET name = $2, sku = $3, description = $4, purpose = $5, sellable = $6,
		     tags = $7, status = $8, updated_at = GREATEST(updated_at, $9)
		 WHERE id = $1
		 RETURNING `+catalogItemColumns,
		id, values.Name, values.SKU, values.Description, values.Purpose, values.Sellable, string(tags), values.Status, updatedAt.UTC(),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domaincatalog.Item{}, domaincatalog.ErrItemNotFound
	}
	if err != nil {
		return domaincatalog.Item{}, fmt.Errorf("update catalog item: %w", err)
	}
	return item, nil
}

// Delete removes one catalog item by UUID.
func (repository *CatalogItemRepository) Delete(ctx context.Context, id string) error {
	result, err := repository.database.ExecContext(ctx, "DELETE FROM catalog_items WHERE id = $1", id)
	if foreignKeyViolation(err, "design_versions_part_fk") {
		return domaincatalog.ErrDesignHistoryExists
	}
	if err != nil {
		return fmt.Errorf("delete catalog item: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted catalog item count: %w", err)
	}
	if rows == 0 {
		return domaincatalog.ErrItemNotFound
	}
	return nil
}

func catalogItemWhere(filter domaincatalog.ListFilter) (string, []any) {
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 5)
	add := func(template string, value any) {
		args = append(args, value)
		conditions = append(conditions, fmt.Sprintf(template, len(args)))
	}
	if filter.Purpose != nil {
		add("purpose = $%d", *filter.Purpose)
	}
	if filter.Status != nil {
		add("status = $%d", *filter.Status)
	}
	if filter.Sellable != nil {
		add("sellable = $%d", *filter.Sellable)
	}
	if filter.Tag != "" {
		add("tags ? $%d", filter.Tag)
	}
	if filter.Query != "" {
		add("(position(lower($%[1]d) in lower(name)) > 0 OR position(lower($%[1]d) in lower(COALESCE(sku, ''))) > 0 OR position(lower($%[1]d) in lower(description)) > 0)", filter.Query)
	}
	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func scanCatalogItem(row rowScanner) (domaincatalog.Item, error) {
	var item domaincatalog.Item
	var sku sql.NullString
	var tags []byte
	if err := row.Scan(
		&item.ID, &item.Name, &sku, &item.Description, &item.Purpose,
		&item.Sellable, &tags, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return domaincatalog.Item{}, err
	}
	if sku.Valid {
		item.SKU = &sku.String
	}
	if err := json.Unmarshal(tags, &item.Tags); err != nil {
		return domaincatalog.Item{}, fmt.Errorf("decode catalog tags: %w", err)
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	item.CreatedAt = item.CreatedAt.UTC()
	item.UpdatedAt = item.UpdatedAt.UTC()
	return item, nil
}
