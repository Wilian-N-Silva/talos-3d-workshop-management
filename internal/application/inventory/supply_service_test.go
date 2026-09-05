package inventory

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
)

const supplyTestID = "11111111-1111-4111-8111-111111111111"
const supplyActorID = "22222222-2222-4222-8222-222222222222"

func TestSupplyServiceNormalizesSupplyAndMovement(t *testing.T) {
	repository := &supplyRepositoryStub{}
	service, err := NewSupplyService(repository)
	if err != nil {
		t.Fatalf("NewSupplyService()=%v", err)
	}
	now := time.Date(2026, 9, 4, 15, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	sku := " NFC-01 "
	supply, err := service.CreateSupply(context.Background(), domain.SupplyValues{Name: " NFC tags ", SKU: &sku, Unit: " unit ", MinimumQuantity: "10.000000", ReplacementUnitCostCents: 75})
	if err != nil || supply.Name != "NFC tags" || repository.supply.MinimumQuantity != "10" || repository.supply.SKU == nil || *repository.supply.SKU != "NFC-01" {
		t.Fatalf("CreateSupply()=%#v,%v values=%#v", supply, err, repository.supply)
	}
	occurred := now.In(time.FixedZone("local", -3*60*60))
	movement, err := service.RecordMovement(context.Background(), supplyTestID, supplyActorID, domain.SupplyMovementValues{Type: domain.SupplyConsume, Quantity: "-2.500000", OccurredAt: occurred})
	if err != nil || movement.Quantity != "-2.5" || repository.actor != supplyActorID || !repository.movement.OccurredAt.Equal(occurred.UTC()) {
		t.Fatalf("RecordMovement()=%#v,%v values=%#v", movement, err, repository.movement)
	}
}

func TestSupplyServiceEnforcesSignedMovementPolicy(t *testing.T) {
	service, _ := NewSupplyService(&supplyRepositoryStub{})
	tests := []domain.SupplyMovementValues{
		{Type: domain.SupplyPurchase, Quantity: "-1", OccurredAt: time.Now()},
		{Type: domain.SupplyConsume, Quantity: "1", OccurredAt: time.Now()},
		{Type: domain.SupplyAdjustment, Quantity: "0", OccurredAt: time.Now()},
		{Type: "unknown", Quantity: "1", OccurredAt: time.Now()},
	}
	for _, input := range tests {
		if _, err := service.RecordMovement(context.Background(), supplyTestID, supplyActorID, input); !errors.Is(err, ErrInvalidSupplyMovement) {
			t.Fatalf("RecordMovement(%#v)=%v", input, err)
		}
	}
}

func TestSupplyServiceDefaultsAndValidatesLowThreshold(t *testing.T) {
	repository := &supplyRepositoryStub{}
	service, _ := NewSupplyService(repository)
	result, err := service.ListLowInventory(context.Background(), "")
	if err != nil || result.SpoolThresholdG != DefaultLowSpoolThresholdG || repository.threshold != DefaultLowSpoolThresholdG {
		t.Fatalf("default threshold result=%#v err=%v repository=%q", result, err, repository.threshold)
	}
	if _, err := service.ListLowInventory(context.Background(), "-1"); !errors.Is(err, ErrInvalidLowThreshold) {
		t.Fatalf("negative threshold=%v", err)
	}
}

type supplyRepositoryStub struct {
	supply    domain.SupplyValues
	movement  domain.SupplyMovementValues
	actor     string
	threshold string
	err       error
}

func (s *supplyRepositoryStub) CreateSupply(_ context.Context, value domain.SupplyValues, _ time.Time) (domain.Supply, error) {
	s.supply = value
	return domain.Supply{Name: value.Name, MinimumQuantity: value.MinimumQuantity}, s.err
}
func (s *supplyRepositoryStub) FindSupply(context.Context, string) (domain.Supply, error) {
	return domain.Supply{}, s.err
}
func (s *supplyRepositoryStub) ListSupplies(context.Context) ([]domain.Supply, error) {
	return []domain.Supply{}, s.err
}
func (s *supplyRepositoryStub) UpdateSupply(context.Context, string, domain.SupplyValues, time.Time) (domain.Supply, error) {
	return domain.Supply{}, s.err
}
func (s *supplyRepositoryStub) DeleteSupply(context.Context, string) error { return s.err }
func (s *supplyRepositoryStub) RecordSupplyMovement(_ context.Context, _ string, actor string, value domain.SupplyMovementValues, _ time.Time) (domain.SupplyMovement, error) {
	s.actor, s.movement = actor, value
	return domain.SupplyMovement{Quantity: value.Quantity}, s.err
}
func (s *supplyRepositoryStub) ListSupplyMovements(context.Context, string) ([]domain.SupplyMovement, error) {
	return []domain.SupplyMovement{}, s.err
}
func (s *supplyRepositoryStub) ListLowInventory(_ context.Context, threshold string) (domain.LowInventory, error) {
	s.threshold = threshold
	return domain.LowInventory{Spools: []domain.Spool{}, Supplies: []domain.Supply{}}, s.err
}
