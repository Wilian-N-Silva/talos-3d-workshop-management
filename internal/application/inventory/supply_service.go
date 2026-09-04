package inventory

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
)

const DefaultLowSpoolThresholdG = "100"

var (
	ErrInvalidSupply         = errors.New("invalid supply")
	ErrInvalidSupplyMovement = errors.New("invalid supply movement")
	ErrInvalidLowThreshold   = errors.New("invalid low inventory threshold")
	quantityPattern          = regexp.MustCompile(`^-?[0-9]{1,12}(\.[0-9]{1,6})?$`)
)

type SupplyRepository interface {
	CreateSupply(context.Context, domain.SupplyValues, time.Time) (domain.Supply, error)
	FindSupply(context.Context, string) (domain.Supply, error)
	ListSupplies(context.Context) ([]domain.Supply, error)
	UpdateSupply(context.Context, string, domain.SupplyValues, time.Time) (domain.Supply, error)
	DeleteSupply(context.Context, string) error
	RecordSupplyMovement(context.Context, string, string, domain.SupplyMovementValues, time.Time) (domain.SupplyMovement, error)
	ListSupplyMovements(context.Context, string) ([]domain.SupplyMovement, error)
	ListLowInventory(context.Context, string) (domain.LowInventory, error)
}

type SupplyService struct {
	repository SupplyRepository
	now        func() time.Time
}

func NewSupplyService(repository SupplyRepository) (*SupplyService, error) {
	if repository == nil {
		return nil, ErrInvalidConfiguration
	}
	return &SupplyService{repository: repository, now: time.Now}, nil
}

func (service *SupplyService) CreateSupply(ctx context.Context, input domain.SupplyValues) (domain.Supply, error) {
	values, err := normalizeSupply(input)
	if err != nil {
		return domain.Supply{}, err
	}
	result, err := service.repository.CreateSupply(ctx, values, service.now().UTC())
	if err != nil {
		return domain.Supply{}, fmt.Errorf("create supply: %w", err)
	}
	return result, nil
}

func (service *SupplyService) GetSupply(ctx context.Context, id string) (domain.Supply, error) {
	if !validID(id) {
		return domain.Supply{}, domain.ErrSupplyNotFound
	}
	result, err := service.repository.FindSupply(ctx, id)
	if err != nil {
		return domain.Supply{}, fmt.Errorf("get supply: %w", err)
	}
	return result, nil
}

func (service *SupplyService) ListSupplies(ctx context.Context) ([]domain.Supply, error) {
	result, err := service.repository.ListSupplies(ctx)
	if err != nil {
		return nil, fmt.Errorf("list supplies: %w", err)
	}
	return result, nil
}

func (service *SupplyService) UpdateSupply(ctx context.Context, id string, input domain.SupplyValues) (domain.Supply, error) {
	if !validID(id) {
		return domain.Supply{}, domain.ErrSupplyNotFound
	}
	values, err := normalizeSupply(input)
	if err != nil {
		return domain.Supply{}, err
	}
	result, err := service.repository.UpdateSupply(ctx, id, values, service.now().UTC())
	if err != nil {
		return domain.Supply{}, fmt.Errorf("update supply: %w", err)
	}
	return result, nil
}

func (service *SupplyService) DeleteSupply(ctx context.Context, id string) error {
	if !validID(id) {
		return domain.ErrSupplyNotFound
	}
	if err := service.repository.DeleteSupply(ctx, id); err != nil {
		return fmt.Errorf("delete supply: %w", err)
	}
	return nil
}

func (service *SupplyService) RecordMovement(ctx context.Context, supplyID, actorID string, input domain.SupplyMovementValues) (domain.SupplyMovement, error) {
	if !validID(supplyID) {
		return domain.SupplyMovement{}, domain.ErrSupplyNotFound
	}
	if !validID(actorID) {
		return domain.SupplyMovement{}, ErrInvalidSupplyMovement
	}
	values, err := normalizeSupplyMovement(input)
	if err != nil {
		return domain.SupplyMovement{}, err
	}
	result, err := service.repository.RecordSupplyMovement(ctx, supplyID, actorID, values, service.now().UTC())
	if err != nil {
		return domain.SupplyMovement{}, fmt.Errorf("record supply movement: %w", err)
	}
	return result, nil
}

func (service *SupplyService) ListMovements(ctx context.Context, supplyID string) ([]domain.SupplyMovement, error) {
	if !validID(supplyID) {
		return nil, domain.ErrSupplyNotFound
	}
	result, err := service.repository.ListSupplyMovements(ctx, supplyID)
	if err != nil {
		return nil, fmt.Errorf("list supply movements: %w", err)
	}
	return result, nil
}

func (service *SupplyService) ListLowInventory(ctx context.Context, spoolThresholdG string) (domain.LowInventory, error) {
	if strings.TrimSpace(spoolThresholdG) == "" {
		spoolThresholdG = DefaultLowSpoolThresholdG
	}
	spoolThresholdG = normalizeDecimal(spoolThresholdG)
	if !validNonnegativeDecimal(spoolThresholdG, weightPattern) {
		return domain.LowInventory{}, ErrInvalidLowThreshold
	}
	result, err := service.repository.ListLowInventory(ctx, spoolThresholdG)
	if err != nil {
		return domain.LowInventory{}, fmt.Errorf("list low inventory: %w", err)
	}
	result.SpoolThresholdG = spoolThresholdG
	return result, nil
}

func normalizeSupply(input domain.SupplyValues) (domain.SupplyValues, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Unit = strings.TrimSpace(input.Unit)
	input.Notes = strings.TrimSpace(input.Notes)
	input.SKU = normalizeOptionalText(input.SKU)
	input.MinimumQuantity = normalizeDecimal(input.MinimumQuantity)
	if input.Name == "" || len(input.Name) > 200 || input.Unit == "" || len(input.Unit) > 50 || len(input.Notes) > 10000 || input.ReplacementUnitCostCents < 0 || !validNonnegativeDecimal(input.MinimumQuantity, quantityPattern) || (input.SKU != nil && len(*input.SKU) > 100) {
		return domain.SupplyValues{}, ErrInvalidSupply
	}
	return input, nil
}

func normalizeSupplyMovement(input domain.SupplyMovementValues) (domain.SupplyMovementValues, error) {
	input.Quantity = normalizeDecimal(input.Quantity)
	input.Notes = strings.TrimSpace(input.Notes)
	input.ReferenceType = normalizeOptionalText(input.ReferenceType)
	input.ReferenceID = normalizeOptionalText(input.ReferenceID)
	if input.OccurredAt.IsZero() || !quantityPattern.MatchString(input.Quantity) || decimal(input.Quantity) == nil || decimal(input.Quantity).Sign() == 0 || len(input.Notes) > 10000 || (input.UnitCostCents != nil && *input.UnitCostCents < 0) || (input.ReferenceType == nil) != (input.ReferenceID == nil) || (input.ReferenceType != nil && len(*input.ReferenceType) > 100) || (input.ReferenceID != nil && len(*input.ReferenceID) > 200) {
		return domain.SupplyMovementValues{}, ErrInvalidSupplyMovement
	}
	input.OccurredAt = input.OccurredAt.UTC()
	sign := decimal(input.Quantity).Sign()
	switch input.Type {
	case domain.SupplyPurchase, domain.SupplyReturn:
		if sign <= 0 {
			return domain.SupplyMovementValues{}, ErrInvalidSupplyMovement
		}
	case domain.SupplyConsume, domain.SupplyDiscard:
		if sign >= 0 {
			return domain.SupplyMovementValues{}, ErrInvalidSupplyMovement
		}
	case domain.SupplyAdjustment:
	default:
		return domain.SupplyMovementValues{}, ErrInvalidSupplyMovement
	}
	return input, nil
}

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}
