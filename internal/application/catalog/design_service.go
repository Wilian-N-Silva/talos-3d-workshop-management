package catalog

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
)

var (
	ErrInvalidPart          = errors.New("invalid catalog part")
	ErrInvalidDesignVersion = errors.New("invalid design version")
	ErrInvalidDesignFile    = errors.New("invalid design file")
)

type DesignRepository interface {
	CreatePart(context.Context, string, domaincatalog.PartValues, time.Time) (domaincatalog.Part, error)
	FindPart(context.Context, string) (domaincatalog.Part, error)
	ListParts(context.Context, string) ([]domaincatalog.Part, error)
	UpdatePart(context.Context, string, domaincatalog.PartValues, time.Time) (domaincatalog.Part, error)
	DeletePart(context.Context, string) error
	CreateDesignVersion(context.Context, string, string, domaincatalog.DesignVersionValues, time.Time) (domaincatalog.DesignVersion, error)
	ListDesignVersions(context.Context, string) ([]domaincatalog.DesignVersion, error)
	AttachDesignFile(context.Context, string, string, string, domaincatalog.DesignFileRole, time.Time) (domaincatalog.DesignFile, error)
}

func (service *DesignService) GetPart(ctx context.Context, partID string) (domaincatalog.Part, error) {
	if !validCatalogID(partID) {
		return domaincatalog.Part{}, domaincatalog.ErrPartNotFound
	}
	part, err := service.repository.FindPart(ctx, partID)
	if err != nil {
		return domaincatalog.Part{}, fmt.Errorf("get catalog part: %w", err)
	}
	return part, nil
}

type DesignService struct {
	repository DesignRepository
	now        func() time.Time
}

func NewDesignService(repository DesignRepository) (*DesignService, error) {
	if repository == nil {
		return nil, ErrInvalidConfiguration
	}
	return &DesignService{repository: repository, now: time.Now}, nil
}

func (service *DesignService) CreatePart(ctx context.Context, itemID string, input domaincatalog.PartValues) (domaincatalog.Part, error) {
	if !validCatalogID(itemID) {
		return domaincatalog.Part{}, domaincatalog.ErrItemNotFound
	}
	values, err := normalizePart(input)
	if err != nil {
		return domaincatalog.Part{}, err
	}
	part, err := service.repository.CreatePart(ctx, itemID, values, service.now().UTC())
	if err != nil {
		return domaincatalog.Part{}, fmt.Errorf("create catalog part: %w", err)
	}
	return part, nil
}

func (service *DesignService) ListParts(ctx context.Context, itemID string) ([]domaincatalog.Part, error) {
	if !validCatalogID(itemID) {
		return nil, domaincatalog.ErrItemNotFound
	}
	parts, err := service.repository.ListParts(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("list catalog parts: %w", err)
	}
	return parts, nil
}

func (service *DesignService) UpdatePart(ctx context.Context, partID string, input domaincatalog.PartValues) (domaincatalog.Part, error) {
	if !validCatalogID(partID) {
		return domaincatalog.Part{}, domaincatalog.ErrPartNotFound
	}
	values, err := normalizePart(input)
	if err != nil {
		return domaincatalog.Part{}, err
	}
	part, err := service.repository.UpdatePart(ctx, partID, values, service.now().UTC())
	if err != nil {
		return domaincatalog.Part{}, fmt.Errorf("update catalog part: %w", err)
	}
	return part, nil
}

func (service *DesignService) DeletePart(ctx context.Context, partID string) error {
	if !validCatalogID(partID) {
		return domaincatalog.ErrPartNotFound
	}
	if err := service.repository.DeletePart(ctx, partID); err != nil {
		return fmt.Errorf("delete catalog part: %w", err)
	}
	return nil
}

func (service *DesignService) CreateVersion(ctx context.Context, partID, actorID string, input domaincatalog.DesignVersionValues) (domaincatalog.DesignVersion, error) {
	if !validCatalogID(partID) {
		return domaincatalog.DesignVersion{}, domaincatalog.ErrPartNotFound
	}
	if !validCatalogID(actorID) {
		return domaincatalog.DesignVersion{}, ErrInvalidDesignVersion
	}
	values, err := normalizeDesignVersion(input)
	if err != nil {
		return domaincatalog.DesignVersion{}, err
	}
	version, err := service.repository.CreateDesignVersion(ctx, partID, actorID, values, service.now().UTC())
	if err != nil {
		return domaincatalog.DesignVersion{}, fmt.Errorf("create design version: %w", err)
	}
	return version, nil
}

func (service *DesignService) ListVersions(ctx context.Context, partID string) ([]domaincatalog.DesignVersion, error) {
	if !validCatalogID(partID) {
		return nil, domaincatalog.ErrPartNotFound
	}
	versions, err := service.repository.ListDesignVersions(ctx, partID)
	if err != nil {
		return nil, fmt.Errorf("list design versions: %w", err)
	}
	return versions, nil
}

func (service *DesignService) AttachFile(ctx context.Context, versionID, fileID, actorID string, role domaincatalog.DesignFileRole) (domaincatalog.DesignFile, error) {
	if !validCatalogID(versionID) || !validCatalogID(fileID) || !validCatalogID(actorID) || !validDesignFileRole(role) {
		return domaincatalog.DesignFile{}, ErrInvalidDesignFile
	}
	file, err := service.repository.AttachDesignFile(ctx, versionID, fileID, actorID, role, service.now().UTC())
	if err != nil {
		return domaincatalog.DesignFile{}, fmt.Errorf("attach design file: %w", err)
	}
	return file, nil
}

func normalizePart(input domaincatalog.PartValues) (domaincatalog.PartValues, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.Name == "" || len(input.Name) > 200 || input.Quantity < 1 || len(input.Notes) > 10000 {
		return domaincatalog.PartValues{}, ErrInvalidPart
	}
	return input, nil
}

func normalizeDesignVersion(input domaincatalog.DesignVersionValues) (domaincatalog.DesignVersionValues, error) {
	input.Version = strings.TrimSpace(input.Version)
	input.Notes = strings.TrimSpace(input.Notes)
	input.OriginalAuthor = strings.TrimSpace(input.OriginalAuthor)
	input.LicenseName = strings.TrimSpace(input.LicenseName)
	input.AttributionText = strings.TrimSpace(input.AttributionText)
	if input.SourceURL != nil {
		value := strings.TrimSpace(*input.SourceURL)
		if value == "" {
			input.SourceURL = nil
		} else {
			parsed, err := url.ParseRequestURI(value)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || len(value) > 2048 {
				return domaincatalog.DesignVersionValues{}, ErrInvalidDesignVersion
			}
			input.SourceURL = &value
		}
	}
	if input.Version == "" || len(input.Version) > 100 || len(input.Notes) > 10000 || len(input.OriginalAuthor) > 200 || len(input.LicenseName) > 200 || len(input.AttributionText) > 4000 || !validDesignOrigin(input.Origin) {
		return domaincatalog.DesignVersionValues{}, ErrInvalidDesignVersion
	}
	if input.AttributionRequired && input.AttributionText == "" {
		return domaincatalog.DesignVersionValues{}, ErrInvalidDesignVersion
	}
	return input, nil
}

func validDesignOrigin(origin domaincatalog.DesignOrigin) bool {
	switch origin {
	case domaincatalog.DesignOriginOriginal, domaincatalog.DesignOriginCustomer, domaincatalog.DesignOriginRemix, domaincatalog.DesignOriginThirdParty, domaincatalog.DesignOriginUnknown:
		return true
	default:
		return false
	}
}

func validDesignFileRole(role domaincatalog.DesignFileRole) bool {
	switch role {
	case domaincatalog.DesignFileSource, domaincatalog.DesignFileMesh, domaincatalog.DesignFilePrint, domaincatalog.DesignFilePreview, domaincatalog.DesignFileDocumentation, domaincatalog.DesignFileOther:
		return true
	default:
		return false
	}
}
