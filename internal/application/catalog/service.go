// Package catalog implements catalog item validation and use cases.
package catalog

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
)

const (
	DefaultListLimit       = 50
	MaximumListLimit       = 100
	maximumNameLength      = 200
	maximumSKULength       = 100
	maximumDescriptionSize = 10000
	maximumTags            = 20
	maximumTagLength       = 50
)

var (
	// ErrInvalidItem indicates invalid user-controlled catalog values.
	ErrInvalidItem = errors.New("invalid catalog item")
	// ErrInvalidListFilter indicates invalid pagination or filter values.
	ErrInvalidListFilter = errors.New("invalid catalog list filter")
	// ErrInvalidConfiguration indicates a missing service dependency.
	ErrInvalidConfiguration = errors.New("invalid catalog service configuration")
	catalogIDPattern        = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// Repository persists catalog items.
type Repository interface {
	Create(context.Context, domaincatalog.Values, time.Time) (domaincatalog.Item, error)
	FindByID(context.Context, string) (domaincatalog.Item, error)
	List(context.Context, domaincatalog.ListFilter) (domaincatalog.Page, error)
	Update(context.Context, string, domaincatalog.Values, time.Time) (domaincatalog.Item, error)
	Delete(context.Context, string) error
}

// Service validates and manages catalog items.
type Service struct {
	repository Repository
	now        func() time.Time
}

// NewService creates a catalog item service.
func NewService(repository Repository) (*Service, error) {
	return newService(repository, time.Now)
}

func newService(repository Repository, now func() time.Time) (*Service, error) {
	if repository == nil || now == nil {
		return nil, ErrInvalidConfiguration
	}
	return &Service{repository: repository, now: now}, nil
}

// Create validates and persists a catalog item.
func (service *Service) Create(ctx context.Context, input domaincatalog.Values) (domaincatalog.Item, error) {
	values, err := normalizeValues(input)
	if err != nil {
		return domaincatalog.Item{}, err
	}
	item, err := service.repository.Create(ctx, values, service.now().UTC())
	if err != nil {
		return domaincatalog.Item{}, fmt.Errorf("create catalog item: %w", err)
	}
	return item, nil
}

// Get returns one catalog item by ID.
func (service *Service) Get(ctx context.Context, id string) (domaincatalog.Item, error) {
	if !validCatalogID(id) {
		return domaincatalog.Item{}, domaincatalog.ErrItemNotFound
	}
	item, err := service.repository.FindByID(ctx, id)
	if err != nil {
		return domaincatalog.Item{}, fmt.Errorf("get catalog item: %w", err)
	}
	return item, nil
}

// List validates filters and returns one catalog page.
func (service *Service) List(ctx context.Context, filter domaincatalog.ListFilter) (domaincatalog.Page, error) {
	normalized, err := normalizeFilter(filter)
	if err != nil {
		return domaincatalog.Page{}, err
	}
	page, err := service.repository.List(ctx, normalized)
	if err != nil {
		return domaincatalog.Page{}, fmt.Errorf("list catalog items: %w", err)
	}
	return page, nil
}

// Update validates and replaces mutable catalog item fields.
func (service *Service) Update(ctx context.Context, id string, input domaincatalog.Values) (domaincatalog.Item, error) {
	if !validCatalogID(id) {
		return domaincatalog.Item{}, domaincatalog.ErrItemNotFound
	}
	values, err := normalizeValues(input)
	if err != nil {
		return domaincatalog.Item{}, err
	}
	item, err := service.repository.Update(ctx, id, values, service.now().UTC())
	if err != nil {
		return domaincatalog.Item{}, fmt.Errorf("update catalog item: %w", err)
	}
	return item, nil
}

// Delete removes one catalog item. Later foreign keys may reject deletion when
// history depends on it; callers can archive an item instead.
func (service *Service) Delete(ctx context.Context, id string) error {
	if !validCatalogID(id) {
		return domaincatalog.ErrItemNotFound
	}
	if err := service.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete catalog item: %w", err)
	}
	return nil
}

func normalizeValues(input domaincatalog.Values) (domaincatalog.Values, error) {
	values := input
	values.Name = strings.TrimSpace(input.Name)
	values.Description = strings.TrimSpace(input.Description)
	if len(values.Name) == 0 || len(values.Name) > maximumNameLength || len(values.Description) > maximumDescriptionSize {
		return domaincatalog.Values{}, ErrInvalidItem
	}
	if input.SKU != nil {
		sku := strings.TrimSpace(*input.SKU)
		if len(sku) > maximumSKULength {
			return domaincatalog.Values{}, ErrInvalidItem
		}
		if sku == "" {
			values.SKU = nil
		} else {
			values.SKU = &sku
		}
	}
	if !validPurpose(values.Purpose) || !validStatus(values.Status) {
		return domaincatalog.Values{}, ErrInvalidItem
	}
	if len(input.Tags) > maximumTags {
		return domaincatalog.Values{}, ErrInvalidItem
	}
	seen := make(map[string]struct{}, len(input.Tags))
	values.Tags = make([]string, 0, len(input.Tags))
	for _, inputTag := range input.Tags {
		tag := strings.ToLower(strings.TrimSpace(inputTag))
		if len(tag) == 0 || len(tag) > maximumTagLength {
			return domaincatalog.Values{}, ErrInvalidItem
		}
		if _, duplicate := seen[tag]; duplicate {
			continue
		}
		seen[tag] = struct{}{}
		values.Tags = append(values.Tags, tag)
	}
	sort.Strings(values.Tags)
	return values, nil
}

func normalizeFilter(filter domaincatalog.ListFilter) (domaincatalog.ListFilter, error) {
	if filter.Limit == 0 {
		filter.Limit = DefaultListLimit
	}
	if filter.Limit < 1 || filter.Limit > MaximumListLimit || filter.Offset < 0 {
		return domaincatalog.ListFilter{}, ErrInvalidListFilter
	}
	if filter.Purpose != nil && !validPurpose(*filter.Purpose) {
		return domaincatalog.ListFilter{}, ErrInvalidListFilter
	}
	if filter.Status != nil && !validStatus(*filter.Status) {
		return domaincatalog.ListFilter{}, ErrInvalidListFilter
	}
	filter.Tag = strings.ToLower(strings.TrimSpace(filter.Tag))
	filter.Query = strings.TrimSpace(filter.Query)
	if len(filter.Tag) > maximumTagLength || len(filter.Query) > maximumNameLength {
		return domaincatalog.ListFilter{}, ErrInvalidListFilter
	}
	return filter, nil
}

func validPurpose(purpose domaincatalog.Purpose) bool {
	switch purpose {
	case domaincatalog.PurposeProduct, domaincatalog.PurposePrototype, domaincatalog.PurposeTooling,
		domaincatalog.PurposeTest, domaincatalog.PurposeInternal, domaincatalog.PurposePersonal:
		return true
	default:
		return false
	}
}

func validStatus(status domaincatalog.Status) bool {
	return status == domaincatalog.StatusActive || status == domaincatalog.StatusArchived
}

func validCatalogID(id string) bool {
	return catalogIDPattern.MatchString(strings.ToLower(strings.TrimSpace(id)))
}
