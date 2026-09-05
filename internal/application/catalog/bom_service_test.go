package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
)

const bomCatalogID = "11111111-1111-4111-8111-111111111111"
const bomItemID = "22222222-2222-4222-8222-222222222222"
const bomSupplyID = "33333333-3333-4333-8333-333333333333"

func TestBOMServiceNormalizesValues(t *testing.T) {
	repository := &bomRepositoryStub{}
	service, err := NewBOMService(repository)
	if err != nil {
		t.Fatalf("NewBOMService()=%v", err)
	}
	now := time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	item, err := service.Create(context.Background(), bomCatalogID, domaincatalog.BOMValues{SupplyID: " " + bomSupplyID + " ", QuantityPerUnit: "1.250000", WastePercent: "10.0000", Notes: " packing "})
	if err != nil || item.QuantityPerUnit != "1.25" || repository.values.WastePercent != "10" || repository.values.Notes != "packing" || !repository.at.Equal(now) {
		t.Fatalf("Create()=%#v,%v values=%#v at=%v", item, err, repository.values, repository.at)
	}
}

func TestBOMPreviewUsesExactDecimalArithmeticWithoutRounding(t *testing.T) {
	inputs := []domaincatalog.BOMCostInput{
		{Item: domaincatalog.BOMItem{QuantityPerUnit: "0.333333", WastePercent: "12.5"}, SupplyName: "NFC", SupplyUnit: "unit", ReplacementUnitCostCents: 99},
		{Item: domaincatalog.BOMItem{QuantityPerUnit: "2", WastePercent: "0"}, SupplyName: "Ring", SupplyUnit: "unit", ReplacementUnitCostCents: 25},
	}
	preview, err := calculateBOMPreview(inputs)
	if err != nil {
		t.Fatalf("calculateBOMPreview()=%v", err)
	}
	if preview.RoundingApplied || preview.Items[0].EffectiveQuantityPerUnit != "0.374999625" || preview.Items[0].ExactReplacementCostCentsPerUnit != "37.124962875" || preview.ExactTotalReplacementCostCents != "87.124962875" {
		t.Fatalf("preview=%#v", preview)
	}
}

func TestBOMServiceRejectsInvalidQuantitiesAndWaste(t *testing.T) {
	service, _ := NewBOMService(&bomRepositoryStub{})
	for _, values := range []domaincatalog.BOMValues{
		{SupplyID: bomSupplyID, QuantityPerUnit: "0", WastePercent: "0"},
		{SupplyID: bomSupplyID, QuantityPerUnit: "1", WastePercent: "-1"},
		{SupplyID: bomSupplyID, QuantityPerUnit: "1.0000001", WastePercent: "0"},
	} {
		if _, err := service.Create(context.Background(), bomCatalogID, values); !errors.Is(err, ErrInvalidBOMItem) {
			t.Fatalf("Create(%#v)=%v", values, err)
		}
	}
}

type bomRepositoryStub struct {
	values domaincatalog.BOMValues
	inputs []domaincatalog.BOMCostInput
	at     time.Time
	err    error
}

func (s *bomRepositoryStub) CreateBOMItem(_ context.Context, catalogID string, values domaincatalog.BOMValues, at time.Time) (domaincatalog.BOMItem, error) {
	s.values, s.at = values, at
	return domaincatalog.BOMItem{CatalogItemID: catalogID, SupplyID: values.SupplyID, QuantityPerUnit: values.QuantityPerUnit, WastePercent: values.WastePercent, Notes: values.Notes}, s.err
}
func (s *bomRepositoryStub) FindBOMItem(context.Context, string, string) (domaincatalog.BOMItem, error) {
	return domaincatalog.BOMItem{}, s.err
}
func (s *bomRepositoryStub) ListBOMCostInputs(context.Context, string) ([]domaincatalog.BOMCostInput, error) {
	return s.inputs, s.err
}
func (s *bomRepositoryStub) UpdateBOMItem(context.Context, string, string, domaincatalog.BOMValues, time.Time) (domaincatalog.BOMItem, error) {
	return domaincatalog.BOMItem{}, s.err
}
func (s *bomRepositoryStub) DeleteBOMItem(context.Context, string, string) error { return s.err }
