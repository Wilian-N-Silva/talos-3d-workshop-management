package inventory

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
)

var (
	ErrInvalidConfiguration = errors.New("invalid inventory service configuration")
	ErrInvalidMaterial      = errors.New("invalid material")
	ErrInvalidSpool         = errors.New("invalid spool")
	ErrInvalidMeasurement   = errors.New("invalid spool measurement")
)

var (
	uuidPattern    = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	densityPattern = regexp.MustCompile(`^[0-9]{1,6}(\.[0-9]{1,6})?$`)
	weightPattern  = regexp.MustCompile(`^[0-9]{1,9}(\.[0-9]{1,3})?$`)
	colorPattern   = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
)

type FilamentRepository interface {
	CreateMaterial(context.Context, domain.MaterialValues, time.Time) (domain.Material, error)
	FindMaterial(context.Context, string) (domain.Material, error)
	ListMaterials(context.Context) ([]domain.Material, error)
	UpdateMaterial(context.Context, string, domain.MaterialValues, time.Time) (domain.Material, error)
	DeleteMaterial(context.Context, string) error
	CreateSpool(context.Context, domain.SpoolValues, time.Time) (domain.Spool, error)
	FindSpool(context.Context, string) (domain.Spool, error)
	ListSpools(context.Context) ([]domain.Spool, error)
	UpdateSpool(context.Context, string, domain.SpoolValues, time.Time) (domain.Spool, error)
	DeleteSpool(context.Context, string) error
	RecordMeasurement(context.Context, string, string, domain.MeasurementValues, time.Time) (domain.SpoolMeasurement, error)
	ListMeasurements(context.Context, string) ([]domain.SpoolMeasurement, error)
}

type FilamentService struct {
	repository FilamentRepository
	now        func() time.Time
}

func NewFilamentService(repository FilamentRepository) (*FilamentService, error) {
	if repository == nil {
		return nil, ErrInvalidConfiguration
	}
	return &FilamentService{repository: repository, now: time.Now}, nil
}

func (service *FilamentService) CreateMaterial(ctx context.Context, input domain.MaterialValues) (domain.Material, error) {
	values, err := normalizeMaterial(input)
	if err != nil {
		return domain.Material{}, err
	}
	result, err := service.repository.CreateMaterial(ctx, values, service.now().UTC())
	if err != nil {
		return domain.Material{}, fmt.Errorf("create material: %w", err)
	}
	return result, nil
}
func (service *FilamentService) GetMaterial(ctx context.Context, id string) (domain.Material, error) {
	if !validID(id) {
		return domain.Material{}, domain.ErrMaterialNotFound
	}
	result, err := service.repository.FindMaterial(ctx, id)
	if err != nil {
		return domain.Material{}, fmt.Errorf("get material: %w", err)
	}
	return result, nil
}
func (service *FilamentService) ListMaterials(ctx context.Context) ([]domain.Material, error) {
	result, err := service.repository.ListMaterials(ctx)
	if err != nil {
		return nil, fmt.Errorf("list materials: %w", err)
	}
	return result, nil
}
func (service *FilamentService) UpdateMaterial(ctx context.Context, id string, input domain.MaterialValues) (domain.Material, error) {
	if !validID(id) {
		return domain.Material{}, domain.ErrMaterialNotFound
	}
	values, err := normalizeMaterial(input)
	if err != nil {
		return domain.Material{}, err
	}
	result, err := service.repository.UpdateMaterial(ctx, id, values, service.now().UTC())
	if err != nil {
		return domain.Material{}, fmt.Errorf("update material: %w", err)
	}
	return result, nil
}
func (service *FilamentService) DeleteMaterial(ctx context.Context, id string) error {
	if !validID(id) {
		return domain.ErrMaterialNotFound
	}
	if err := service.repository.DeleteMaterial(ctx, id); err != nil {
		return fmt.Errorf("delete material: %w", err)
	}
	return nil
}
func (service *FilamentService) CreateSpool(ctx context.Context, input domain.SpoolValues) (domain.Spool, error) {
	values, err := normalizeSpool(input)
	if err != nil {
		return domain.Spool{}, err
	}
	result, err := service.repository.CreateSpool(ctx, values, service.now().UTC())
	if err != nil {
		return domain.Spool{}, fmt.Errorf("create spool: %w", err)
	}
	return result, nil
}
func (service *FilamentService) GetSpool(ctx context.Context, id string) (domain.Spool, error) {
	if !validID(id) {
		return domain.Spool{}, domain.ErrSpoolNotFound
	}
	result, err := service.repository.FindSpool(ctx, id)
	if err != nil {
		return domain.Spool{}, fmt.Errorf("get spool: %w", err)
	}
	return result, nil
}
func (service *FilamentService) ListSpools(ctx context.Context) ([]domain.Spool, error) {
	result, err := service.repository.ListSpools(ctx)
	if err != nil {
		return nil, fmt.Errorf("list spools: %w", err)
	}
	return result, nil
}
func (service *FilamentService) UpdateSpool(ctx context.Context, id string, input domain.SpoolValues) (domain.Spool, error) {
	if !validID(id) {
		return domain.Spool{}, domain.ErrSpoolNotFound
	}
	values, err := normalizeSpool(input)
	if err != nil {
		return domain.Spool{}, err
	}
	result, err := service.repository.UpdateSpool(ctx, id, values, service.now().UTC())
	if err != nil {
		return domain.Spool{}, fmt.Errorf("update spool: %w", err)
	}
	return result, nil
}
func (service *FilamentService) DeleteSpool(ctx context.Context, id string) error {
	if !validID(id) {
		return domain.ErrSpoolNotFound
	}
	if err := service.repository.DeleteSpool(ctx, id); err != nil {
		return fmt.Errorf("delete spool: %w", err)
	}
	return nil
}
func (service *FilamentService) RecordMeasurement(ctx context.Context, spoolID, actorID string, input domain.MeasurementValues) (domain.SpoolMeasurement, error) {
	if !validID(spoolID) {
		return domain.SpoolMeasurement{}, domain.ErrSpoolNotFound
	}
	if !validID(actorID) {
		return domain.SpoolMeasurement{}, ErrInvalidMeasurement
	}
	values, err := normalizeMeasurement(input)
	if err != nil {
		return domain.SpoolMeasurement{}, err
	}
	result, err := service.repository.RecordMeasurement(ctx, spoolID, actorID, values, service.now().UTC())
	if err != nil {
		return domain.SpoolMeasurement{}, fmt.Errorf("record spool measurement: %w", err)
	}
	return result, nil
}
func (service *FilamentService) ListMeasurements(ctx context.Context, spoolID string) ([]domain.SpoolMeasurement, error) {
	if !validID(spoolID) {
		return nil, domain.ErrSpoolNotFound
	}
	result, err := service.repository.ListMeasurements(ctx, spoolID)
	if err != nil {
		return nil, fmt.Errorf("list spool measurements: %w", err)
	}
	return result, nil
}

func normalizeMaterial(input domain.MaterialValues) (domain.MaterialValues, error) {
	input.Manufacturer, input.Name, input.MaterialType = strings.TrimSpace(input.Manufacturer), strings.TrimSpace(input.Name), strings.TrimSpace(input.MaterialType)
	input.ColorName, input.Notes = strings.TrimSpace(input.ColorName), strings.TrimSpace(input.Notes)
	input.NominalDensity = normalizeDecimal(input.NominalDensity)
	if input.ColorHex != nil {
		value := strings.ToUpper(strings.TrimSpace(*input.ColorHex))
		if value == "" {
			input.ColorHex = nil
		} else {
			input.ColorHex = &value
		}
	}
	if input.Manufacturer == "" || len(input.Manufacturer) > 200 || input.Name == "" || len(input.Name) > 200 || input.MaterialType == "" || len(input.MaterialType) > 100 || len(input.ColorName) > 100 || len(input.Notes) > 10000 || input.DefaultReplacementCostPerKgCents < 0 || !validPositiveDecimal(input.NominalDensity, densityPattern) || (input.ColorHex != nil && !colorPattern.MatchString(*input.ColorHex)) {
		return domain.MaterialValues{}, ErrInvalidMaterial
	}
	return input, nil
}

func normalizeSpool(input domain.SpoolValues) (domain.SpoolValues, error) {
	input.Code, input.StorageLocation, input.StorageStatus, input.LotNumber = strings.TrimSpace(input.Code), strings.TrimSpace(input.StorageLocation), strings.TrimSpace(input.StorageStatus), strings.TrimSpace(input.LotNumber)
	input.NominalNetWeightG, input.TareWeightG = normalizeDecimal(input.NominalNetWeightG), normalizeDecimal(input.TareWeightG)
	if input.GrossWeightAtOpenG != nil {
		value := normalizeDecimal(*input.GrossWeightAtOpenG)
		if value == "" {
			input.GrossWeightAtOpenG = nil
		} else {
			input.GrossWeightAtOpenG = &value
		}
	}
	if input.Code == "" || len(input.Code) > 100 || !validID(input.MaterialID) || !validPositiveDecimal(input.NominalNetWeightG, weightPattern) || !validNonnegativeDecimal(input.TareWeightG, weightPattern) || input.PurchaseCostCents < 0 || input.ReplacementCostPerKgCents < 0 || len(input.StorageLocation) > 200 || len(input.StorageStatus) > 100 || len(input.LotNumber) > 200 || !validSpoolStatus(input.Status) {
		return domain.SpoolValues{}, ErrInvalidSpool
	}
	if input.GrossWeightAtOpenG != nil && (!validNonnegativeDecimal(*input.GrossWeightAtOpenG, weightPattern) || compareDecimals(*input.GrossWeightAtOpenG, input.TareWeightG) < 0) {
		return domain.SpoolValues{}, ErrInvalidSpool
	}
	return input, nil
}

func normalizeMeasurement(input domain.MeasurementValues) (domain.MeasurementValues, error) {
	input.GrossWeightG, input.Notes = normalizeDecimal(input.GrossWeightG), strings.TrimSpace(input.Notes)
	if input.MeasuredAt.IsZero() {
		return domain.MeasurementValues{}, ErrInvalidMeasurement
	}
	input.MeasuredAt = input.MeasuredAt.UTC()
	if !validNonnegativeDecimal(input.GrossWeightG, weightPattern) || len(input.Notes) > 10000 || !validMeasurementSource(input.Source) {
		return domain.MeasurementValues{}, ErrInvalidMeasurement
	}
	return input, nil
}

func normalizeDecimal(value string) string {
	value = strings.TrimSpace(value)
	if strings.Contains(value, ".") {
		value = strings.TrimRight(strings.TrimRight(value, "0"), ".")
	}
	if value == "" {
		return "0"
	}
	return value
}
func decimal(value string) *big.Rat {
	result, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil
	}
	return result
}
func validPositiveDecimal(value string, pattern *regexp.Regexp) bool {
	parsed := decimal(value)
	return pattern.MatchString(value) && parsed != nil && parsed.Sign() > 0
}
func validNonnegativeDecimal(value string, pattern *regexp.Regexp) bool {
	parsed := decimal(value)
	return pattern.MatchString(value) && parsed != nil && parsed.Sign() >= 0
}
func compareDecimals(left, right string) int { return decimal(left).Cmp(decimal(right)) }
func validID(value string) bool {
	return uuidPattern.MatchString(strings.ToLower(strings.TrimSpace(value)))
}
func validSpoolStatus(value domain.SpoolStatus) bool {
	switch value {
	case domain.SpoolSealed, domain.SpoolOpen, domain.SpoolStored, domain.SpoolDrying, domain.SpoolEmpty, domain.SpoolRetired:
		return true
	}
	return false
}
func validMeasurementSource(value domain.MeasurementSource) bool {
	return value == domain.MeasurementManual || value == domain.MeasurementImported || value == domain.MeasurementOther
}
