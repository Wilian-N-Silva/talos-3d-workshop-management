package httpplatform

import (
	"context"
	"net/http"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
	"github.com/go-chi/chi/v5"
)

const (
	SuppliesPath     = "/inventory/supplies"
	LowInventoryPath = "/inventory/low-stock"
)

type SupplyInventoryService interface {
	CreateSupply(context.Context, domain.SupplyValues) (domain.Supply, error)
	GetSupply(context.Context, string) (domain.Supply, error)
	ListSupplies(context.Context) ([]domain.Supply, error)
	UpdateSupply(context.Context, string, domain.SupplyValues) (domain.Supply, error)
	DeleteSupply(context.Context, string) error
	RecordMovement(context.Context, string, string, domain.SupplyMovementValues) (domain.SupplyMovement, error)
	ListMovements(context.Context, string) ([]domain.SupplyMovement, error)
	ListLowInventory(context.Context, string) (domain.LowInventory, error)
}

type supplyRequest struct {
	Name                     string  `json:"name"`
	SKU                      *string `json:"sku"`
	Unit                     string  `json:"unit"`
	ReplacementUnitCostCents int64   `json:"replacement_unit_cost_cents"`
	MinimumQuantity          string  `json:"minimum_quantity"`
	Notes                    string  `json:"notes"`
}

type supplyMovementRequest struct {
	Type          domain.SupplyMovementType `json:"type"`
	Quantity      string                    `json:"quantity"`
	UnitCostCents *int64                    `json:"unit_cost_cents"`
	ReferenceType *string                   `json:"reference_type"`
	ReferenceID   *string                   `json:"reference_id"`
	OccurredAt    time.Time                 `json:"occurred_at"`
	Notes         string                    `json:"notes"`
}

func RegisterSupplyInventory(router *APIV1Router, authentication BearerAuthenticationService, service SupplyInventoryService) {
	supplyPath := SuppliesPath + "/{supplyID}"
	movementsPath := supplyPath + "/movements"

	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryRead, http.MethodGet, SuppliesPath, func(w http.ResponseWriter, r *http.Request) {
		values, err := service.ListSupplies(r.Context())
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"supplies": values})
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryWrite, http.MethodPost, SuppliesPath, func(w http.ResponseWriter, r *http.Request) {
		var body supplyRequest
		if !decodeInventory(w, r, &body) {
			return
		}
		value, err := service.CreateSupply(r.Context(), body.values())
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryRead, http.MethodGet, supplyPath, func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetSupply(r.Context(), chi.URLParam(r, "supplyID"))
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryWrite, http.MethodPut, supplyPath, func(w http.ResponseWriter, r *http.Request) {
		var body supplyRequest
		if !decodeInventory(w, r, &body) {
			return
		}
		value, err := service.UpdateSupply(r.Context(), chi.URLParam(r, "supplyID"), body.values())
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryWrite, http.MethodDelete, supplyPath, func(w http.ResponseWriter, r *http.Request) {
		if err := service.DeleteSupply(r.Context(), chi.URLParam(r, "supplyID")); err != nil {
			writeInventoryError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryRead, http.MethodGet, movementsPath, func(w http.ResponseWriter, r *http.Request) {
		values, err := service.ListMovements(r.Context(), chi.URLParam(r, "supplyID"))
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"movements": values})
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryWrite, http.MethodPost, movementsPath, func(w http.ResponseWriter, r *http.Request) {
		var body supplyMovementRequest
		if !decodeInventory(w, r, &body) {
			return
		}
		user, ok := CurrentUserFromContext(r.Context())
		if !ok {
			writeUnauthenticated(w)
			return
		}
		value, err := service.RecordMovement(r.Context(), chi.URLParam(r, "supplyID"), user.ID, body.values())
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryRead, http.MethodGet, LowInventoryPath, func(w http.ResponseWriter, r *http.Request) {
		value, err := service.ListLowInventory(r.Context(), r.URL.Query().Get("spool_threshold_g"))
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
}

func (request supplyRequest) values() domain.SupplyValues {
	return domain.SupplyValues{
		Name:                     request.Name,
		SKU:                      request.SKU,
		Unit:                     request.Unit,
		ReplacementUnitCostCents: request.ReplacementUnitCostCents,
		MinimumQuantity:          request.MinimumQuantity,
		Notes:                    request.Notes,
	}
}

func (request supplyMovementRequest) values() domain.SupplyMovementValues {
	return domain.SupplyMovementValues{
		Type:          request.Type,
		Quantity:      request.Quantity,
		UnitCostCents: request.UnitCostCents,
		ReferenceType: request.ReferenceType,
		ReferenceID:   request.ReferenceID,
		OccurredAt:    request.OccurredAt,
		Notes:         request.Notes,
	}
}
