package catalog

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
)

var (
	ErrInvalidBOMItem      = errors.New("invalid catalog BOM item")
	ErrInvalidBOMService   = errors.New("invalid catalog BOM service configuration")
	bomQuantityPattern     = regexp.MustCompile(`^[0-9]{1,12}(\.[0-9]{1,6})?$`)
	bomWastePercentPattern = regexp.MustCompile(`^[0-9]{1,5}(\.[0-9]{1,4})?$`)
)

type BOMRepository interface {
	CreateBOMItem(context.Context, string, domaincatalog.BOMValues, time.Time) (domaincatalog.BOMItem, error)
	FindBOMItem(context.Context, string, string) (domaincatalog.BOMItem, error)
	ListBOMCostInputs(context.Context, string) ([]domaincatalog.BOMCostInput, error)
	UpdateBOMItem(context.Context, string, string, domaincatalog.BOMValues, time.Time) (domaincatalog.BOMItem, error)
	DeleteBOMItem(context.Context, string, string) error
}

type BOMService struct {
	repository BOMRepository
	now        func() time.Time
}

func NewBOMService(repository BOMRepository) (*BOMService, error) {
	if repository == nil {
		return nil, ErrInvalidBOMService
	}
	return &BOMService{repository: repository, now: time.Now}, nil
}

func (service *BOMService) Create(ctx context.Context, catalogItemID string, input domaincatalog.BOMValues) (domaincatalog.BOMItem, error) {
	if !validCatalogID(catalogItemID) {
		return domaincatalog.BOMItem{}, domaincatalog.ErrItemNotFound
	}
	values, err := normalizeBOMValues(input)
	if err != nil {
		return domaincatalog.BOMItem{}, err
	}
	result, err := service.repository.CreateBOMItem(ctx, catalogItemID, values, service.now().UTC())
	if err != nil {
		return domaincatalog.BOMItem{}, fmt.Errorf("create catalog BOM item: %w", err)
	}
	return result, nil
}

func (service *BOMService) Get(ctx context.Context, catalogItemID, bomItemID string) (domaincatalog.BOMItem, error) {
	if !validCatalogID(catalogItemID) {
		return domaincatalog.BOMItem{}, domaincatalog.ErrItemNotFound
	}
	if !validCatalogID(bomItemID) {
		return domaincatalog.BOMItem{}, domaincatalog.ErrBOMItemNotFound
	}
	result, err := service.repository.FindBOMItem(ctx, catalogItemID, bomItemID)
	if err != nil {
		return domaincatalog.BOMItem{}, fmt.Errorf("get catalog BOM item: %w", err)
	}
	return result, nil
}

func (service *BOMService) Preview(ctx context.Context, catalogItemID string) (domaincatalog.BOMPreview, error) {
	if !validCatalogID(catalogItemID) {
		return domaincatalog.BOMPreview{}, domaincatalog.ErrItemNotFound
	}
	inputs, err := service.repository.ListBOMCostInputs(ctx, catalogItemID)
	if err != nil {
		return domaincatalog.BOMPreview{}, fmt.Errorf("list catalog BOM: %w", err)
	}
	return calculateBOMPreview(inputs)
}

func (service *BOMService) Update(ctx context.Context, catalogItemID, bomItemID string, input domaincatalog.BOMValues) (domaincatalog.BOMItem, error) {
	if !validCatalogID(catalogItemID) {
		return domaincatalog.BOMItem{}, domaincatalog.ErrItemNotFound
	}
	if !validCatalogID(bomItemID) {
		return domaincatalog.BOMItem{}, domaincatalog.ErrBOMItemNotFound
	}
	values, err := normalizeBOMValues(input)
	if err != nil {
		return domaincatalog.BOMItem{}, err
	}
	result, err := service.repository.UpdateBOMItem(ctx, catalogItemID, bomItemID, values, service.now().UTC())
	if err != nil {
		return domaincatalog.BOMItem{}, fmt.Errorf("update catalog BOM item: %w", err)
	}
	return result, nil
}

func (service *BOMService) Delete(ctx context.Context, catalogItemID, bomItemID string) error {
	if !validCatalogID(catalogItemID) {
		return domaincatalog.ErrItemNotFound
	}
	if !validCatalogID(bomItemID) {
		return domaincatalog.ErrBOMItemNotFound
	}
	if err := service.repository.DeleteBOMItem(ctx, catalogItemID, bomItemID); err != nil {
		return fmt.Errorf("delete catalog BOM item: %w", err)
	}
	return nil
}

func normalizeBOMValues(input domaincatalog.BOMValues) (domaincatalog.BOMValues, error) {
	input.SupplyID = strings.ToLower(strings.TrimSpace(input.SupplyID))
	input.QuantityPerUnit = normalizeBOMDecimal(input.QuantityPerUnit)
	input.WastePercent = normalizeBOMDecimal(input.WastePercent)
	input.Notes = strings.TrimSpace(input.Notes)
	quantity, quantityOK := new(big.Rat).SetString(input.QuantityPerUnit)
	waste, wasteOK := new(big.Rat).SetString(input.WastePercent)
	if !validCatalogID(input.SupplyID) || !bomQuantityPattern.MatchString(input.QuantityPerUnit) || !quantityOK || quantity.Sign() <= 0 || !bomWastePercentPattern.MatchString(input.WastePercent) || !wasteOK || waste.Sign() < 0 || len(input.Notes) > 10000 {
		return domaincatalog.BOMValues{}, ErrInvalidBOMItem
	}
	return input, nil
}

func calculateBOMPreview(inputs []domaincatalog.BOMCostInput) (domaincatalog.BOMPreview, error) {
	preview := domaincatalog.BOMPreview{Items: make([]domaincatalog.BOMPreviewLine, 0, len(inputs)), RoundingApplied: false}
	total := new(big.Rat)
	for _, input := range inputs {
		quantity, quantityOK := new(big.Rat).SetString(input.Item.QuantityPerUnit)
		waste, wasteOK := new(big.Rat).SetString(input.Item.WastePercent)
		if !quantityOK || !wasteOK || quantity.Sign() <= 0 || waste.Sign() < 0 || input.ReplacementUnitCostCents < 0 {
			return domaincatalog.BOMPreview{}, errors.New("invalid persisted catalog BOM cost input")
		}
		factor := new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).Quo(waste, big.NewRat(100, 1)))
		effectiveQuantity := new(big.Rat).Mul(quantity, factor)
		exactCost := new(big.Rat).Mul(effectiveQuantity, big.NewRat(input.ReplacementUnitCostCents, 1))
		total.Add(total, exactCost)
		preview.Items = append(preview.Items, domaincatalog.BOMPreviewLine{
			Item:                             input.Item,
			SupplyName:                       input.SupplyName,
			SupplyUnit:                       input.SupplyUnit,
			ReplacementUnitCostCents:         input.ReplacementUnitCostCents,
			EffectiveQuantityPerUnit:         formatBOMDecimal(effectiveQuantity),
			ExactReplacementCostCentsPerUnit: formatBOMDecimal(exactCost),
		})
	}
	preview.ExactTotalReplacementCostCents = formatBOMDecimal(total)
	return preview, nil
}

func normalizeBOMDecimal(value string) string {
	value = strings.TrimSpace(value)
	if strings.Contains(value, ".") {
		value = strings.TrimRight(strings.TrimRight(value, "0"), ".")
	}
	if value == "" {
		return "0"
	}
	return value
}

func formatBOMDecimal(value *big.Rat) string {
	return normalizeBOMDecimal(value.FloatString(12))
}
