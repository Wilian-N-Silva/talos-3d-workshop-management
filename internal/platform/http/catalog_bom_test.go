package httpplatform

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	applicationcatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/catalog"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
)

func TestCatalogBOMRoutesReturnExactPreviewAndEnforcePermissions(t *testing.T) {
	now := time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC)
	item := domaincatalog.BOMItem{ID: "bom-id", CatalogItemID: "item-id", SupplyID: "supply-id", QuantityPerUnit: "1", WastePercent: "10", CreatedAt: now, UpdatedAt: now}
	service := &catalogBOMServiceStub{item: item, preview: domaincatalog.BOMPreview{Items: []domaincatalog.BOMPreviewLine{{Item: item, SupplyName: "NFC", SupplyUnit: "unit", ReplacementUnitCostCents: 75, EffectiveQuantityPerUnit: "1.1", ExactReplacementCostCentsPerUnit: "82.5"}}, ExactTotalReplacementCostCents: "82.5"}}
	router := NewAPIV1Router()
	RegisterCatalogBOM(router, authorizedCatalogUser(domainauth.RoleOwner), service)

	preview := inventoryRequest(router, http.MethodGet, "/catalog/items/item-id/bom", "")
	if preview.Code != http.StatusOK || service.catalogItemID != "item-id" || !strings.Contains(preview.Body.String(), `"exact_replacement_cost_cents_per_unit":"82.5"`) || !strings.Contains(preview.Body.String(), `"rounding_applied":false`) {
		t.Fatalf("preview=%d item=%q body=%s", preview.Code, service.catalogItemID, preview.Body.String())
	}
	created := inventoryRequest(router, http.MethodPost, "/catalog/items/item-id/bom", `{"supply_id":"supply-id","quantity_per_unit":"1.000000","waste_percent":"10.0000","notes":"tag"}`)
	if created.Code != http.StatusCreated || service.values.QuantityPerUnit != "1.000000" {
		t.Fatalf("create=%d values=%#v body=%s", created.Code, service.values, created.Body.String())
	}

	viewer := NewAPIV1Router()
	RegisterCatalogBOM(viewer, authorizedCatalogUser(domainauth.RoleViewer), service)
	forbidden := inventoryRequest(viewer, http.MethodPost, "/catalog/items/item-id/bom", `{"supply_id":"supply-id","quantity_per_unit":"1","waste_percent":"0"}`)
	assertAPIError(t, forbidden, http.StatusForbidden, "forbidden", "Permission denied")
}

func TestCatalogBOMErrorsAreStable(t *testing.T) {
	service := &catalogBOMServiceStub{err: domaincatalog.ErrBOMSupplyConflict}
	router := NewAPIV1Router()
	RegisterCatalogBOM(router, authorizedCatalogUser(domainauth.RoleOwner), service)
	response := inventoryRequest(router, http.MethodPost, "/catalog/items/item-id/bom", `{"supply_id":"supply-id","quantity_per_unit":"1","waste_percent":"0"}`)
	assertAPIError(t, response, http.StatusConflict, "catalog_bom_supply_exists", "Supply already exists in catalog BOM")

	service.err = applicationcatalog.ErrInvalidBOMItem
	response = inventoryRequest(router, http.MethodPost, "/catalog/items/item-id/bom", `{"supply_id":"supply-id","quantity_per_unit":"0","waste_percent":"0"}`)
	assertAPIError(t, response, http.StatusBadRequest, "invalid_catalog_bom_item", "Invalid catalog BOM item")
}

type catalogBOMServiceStub struct {
	item          domaincatalog.BOMItem
	preview       domaincatalog.BOMPreview
	values        domaincatalog.BOMValues
	catalogItemID string
	bomItemID     string
	err           error
}

func (s *catalogBOMServiceStub) Create(_ context.Context, catalogItemID string, values domaincatalog.BOMValues) (domaincatalog.BOMItem, error) {
	s.catalogItemID, s.values = catalogItemID, values
	return s.item, s.err
}
func (s *catalogBOMServiceStub) Get(_ context.Context, catalogItemID, bomItemID string) (domaincatalog.BOMItem, error) {
	s.catalogItemID, s.bomItemID = catalogItemID, bomItemID
	return s.item, s.err
}
func (s *catalogBOMServiceStub) Preview(_ context.Context, catalogItemID string) (domaincatalog.BOMPreview, error) {
	s.catalogItemID = catalogItemID
	return s.preview, s.err
}
func (s *catalogBOMServiceStub) Update(_ context.Context, catalogItemID, bomItemID string, values domaincatalog.BOMValues) (domaincatalog.BOMItem, error) {
	s.catalogItemID, s.bomItemID, s.values = catalogItemID, bomItemID, values
	return s.item, s.err
}
func (s *catalogBOMServiceStub) Delete(_ context.Context, catalogItemID, bomItemID string) error {
	s.catalogItemID, s.bomItemID = catalogItemID, bomItemID
	return s.err
}
