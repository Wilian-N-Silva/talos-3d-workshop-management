package httpplatform

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
)

func TestSupplyRoutesEnforcePermissionsAndRecordActor(t *testing.T) {
	now := time.Date(2026, 9, 4, 15, 0, 0, 0, time.UTC)
	service := &supplyServiceStub{
		supply:   domain.Supply{ID: "supply-id", Name: "NFC", Unit: "unit", CurrentQuantity: "0.000000", MinimumQuantity: "10.000000", CreatedAt: now, UpdatedAt: now},
		movement: domain.SupplyMovement{ID: "movement-id", SupplyID: "supply-id", Type: domain.SupplyPurchase, Quantity: "20.000000", OccurredAt: now, RecordedBy: "user-id", CreatedAt: now},
	}
	router := NewAPIV1Router()
	RegisterSupplyInventory(router, authorizedCatalogUser(domainauth.RoleOperator), service)

	created := inventoryRequest(router, http.MethodPost, SuppliesPath, `{"name":"NFC","unit":"unit","replacement_unit_cost_cents":75,"minimum_quantity":"10.000000","notes":"tags"}`)
	if created.Code != http.StatusCreated || service.supplyValues.MinimumQuantity != "10.000000" || !strings.Contains(created.Body.String(), `"current_quantity":"0.000000"`) {
		t.Fatalf("create supply=%d values=%#v body=%s", created.Code, service.supplyValues, created.Body.String())
	}
	moved := inventoryRequest(router, http.MethodPost, SuppliesPath+"/supply-id/movements", `{"type":"purchase","quantity":"20.000000","occurred_at":"2026-09-04T15:00:00Z"}`)
	if moved.Code != http.StatusCreated || service.supplyID != "supply-id" || service.actorID != "user-id" || service.movementValues.Quantity != "20.000000" {
		t.Fatalf("movement=%d supply/actor=%q/%q values=%#v body=%s", moved.Code, service.supplyID, service.actorID, service.movementValues, moved.Body.String())
	}

	viewer := NewAPIV1Router()
	RegisterSupplyInventory(viewer, authorizedCatalogUser(domainauth.RoleViewer), service)
	forbidden := inventoryRequest(viewer, http.MethodPost, SuppliesPath, `{"name":"NFC","unit":"unit","minimum_quantity":"0"}`)
	assertAPIError(t, forbidden, http.StatusForbidden, "forbidden", "Permission denied")
}

func TestLowInventoryRoutePassesConfigurableThreshold(t *testing.T) {
	service := &supplyServiceStub{low: domain.LowInventory{SpoolThresholdG: "75", Spools: []domain.Spool{}, Supplies: []domain.Supply{}}}
	router := NewAPIV1Router()
	RegisterSupplyInventory(router, authorizedCatalogUser(domainauth.RoleViewer), service)
	response := inventoryRequest(router, http.MethodGet, LowInventoryPath+"?spool_threshold_g=75", "")
	if response.Code != http.StatusOK || service.threshold != "75" || !strings.Contains(response.Body.String(), `"spool_threshold_g":"75"`) {
		t.Fatalf("low inventory=%d threshold=%q body=%s", response.Code, service.threshold, response.Body.String())
	}
}

func TestSupplyInventoryErrorsAreStable(t *testing.T) {
	service := &supplyServiceStub{err: domain.ErrInsufficientStock}
	router := NewAPIV1Router()
	RegisterSupplyInventory(router, authorizedCatalogUser(domainauth.RoleOperator), service)
	response := inventoryRequest(router, http.MethodPost, SuppliesPath+"/supply-id/movements", `{"type":"consume","quantity":"-5","occurred_at":"2026-09-04T15:00:00Z"}`)
	assertAPIError(t, response, http.StatusConflict, "insufficient_stock", "Movement would make supply stock negative")
}

type supplyServiceStub struct {
	supply         domain.Supply
	movement       domain.SupplyMovement
	low            domain.LowInventory
	supplyValues   domain.SupplyValues
	movementValues domain.SupplyMovementValues
	supplyID       string
	actorID        string
	threshold      string
	err            error
}

func (s *supplyServiceStub) CreateSupply(_ context.Context, values domain.SupplyValues) (domain.Supply, error) {
	s.supplyValues = values
	return s.supply, s.err
}
func (s *supplyServiceStub) GetSupply(context.Context, string) (domain.Supply, error) {
	return s.supply, s.err
}
func (s *supplyServiceStub) ListSupplies(context.Context) ([]domain.Supply, error) {
	return []domain.Supply{s.supply}, s.err
}
func (s *supplyServiceStub) UpdateSupply(_ context.Context, _ string, values domain.SupplyValues) (domain.Supply, error) {
	s.supplyValues = values
	return s.supply, s.err
}
func (s *supplyServiceStub) DeleteSupply(context.Context, string) error { return s.err }
func (s *supplyServiceStub) RecordMovement(_ context.Context, supplyID, actorID string, values domain.SupplyMovementValues) (domain.SupplyMovement, error) {
	s.supplyID, s.actorID, s.movementValues = supplyID, actorID, values
	return s.movement, s.err
}
func (s *supplyServiceStub) ListMovements(_ context.Context, supplyID string) ([]domain.SupplyMovement, error) {
	s.supplyID = supplyID
	return []domain.SupplyMovement{s.movement}, s.err
}
func (s *supplyServiceStub) ListLowInventory(_ context.Context, threshold string) (domain.LowInventory, error) {
	s.threshold = threshold
	return s.low, s.err
}
