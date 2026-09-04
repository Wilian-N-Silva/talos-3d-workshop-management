package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	applicationcatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/catalog"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
	domaininventory "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
	"github.com/go-chi/chi/v5"
)

const CatalogBOMPath = CatalogItemsPath + "/{itemID}/bom"

type CatalogBOMService interface {
	Create(context.Context, string, domaincatalog.BOMValues) (domaincatalog.BOMItem, error)
	Get(context.Context, string, string) (domaincatalog.BOMItem, error)
	Preview(context.Context, string) (domaincatalog.BOMPreview, error)
	Update(context.Context, string, string, domaincatalog.BOMValues) (domaincatalog.BOMItem, error)
	Delete(context.Context, string, string) error
}

type catalogBOMRequest struct {
	SupplyID        string `json:"supply_id"`
	QuantityPerUnit string `json:"quantity_per_unit"`
	WastePercent    string `json:"waste_percent"`
	Notes           string `json:"notes"`
}

type catalogBOMItemResponse struct {
	ID              string    `json:"id"`
	CatalogItemID   string    `json:"catalog_item_id"`
	SupplyID        string    `json:"supply_id"`
	QuantityPerUnit string    `json:"quantity_per_unit"`
	WastePercent    string    `json:"waste_percent"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type catalogBOMPreviewLineResponse struct {
	catalogBOMItemResponse
	SupplyName                       string `json:"supply_name"`
	SupplyUnit                       string `json:"supply_unit"`
	ReplacementUnitCostCents         int64  `json:"replacement_unit_cost_cents"`
	EffectiveQuantityPerUnit         string `json:"effective_quantity_per_unit"`
	ExactReplacementCostCentsPerUnit string `json:"exact_replacement_cost_cents_per_unit"`
}

type catalogBOMPreviewResponse struct {
	Items                          []catalogBOMPreviewLineResponse `json:"items"`
	ExactTotalReplacementCostCents string                          `json:"exact_total_replacement_cost_cents"`
	RoundingApplied                bool                            `json:"rounding_applied"`
}

func RegisterCatalogBOM(router *APIV1Router, authentication BearerAuthenticationService, service CatalogBOMService) {
	bomItemPath := CatalogBOMPath + "/{bomItemID}"
	registerCatalogBOMRoute(router, authentication, domainauth.PermissionCatalogRead, http.MethodGet, CatalogBOMPath, func(w http.ResponseWriter, r *http.Request) {
		preview, err := service.Preview(r.Context(), chi.URLParam(r, "itemID"))
		if err != nil {
			writeCatalogBOMError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, newCatalogBOMPreviewResponse(preview))
	})
	registerCatalogBOMRoute(router, authentication, domainauth.PermissionCatalogWrite, http.MethodPost, CatalogBOMPath, func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeCatalogBOMRequest(w, r)
		if !ok {
			return
		}
		item, err := service.Create(r.Context(), chi.URLParam(r, "itemID"), body.values())
		if err != nil {
			writeCatalogBOMError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, newCatalogBOMItemResponse(item))
	})
	registerCatalogBOMRoute(router, authentication, domainauth.PermissionCatalogRead, http.MethodGet, bomItemPath, func(w http.ResponseWriter, r *http.Request) {
		item, err := service.Get(r.Context(), chi.URLParam(r, "itemID"), chi.URLParam(r, "bomItemID"))
		if err != nil {
			writeCatalogBOMError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, newCatalogBOMItemResponse(item))
	})
	registerCatalogBOMRoute(router, authentication, domainauth.PermissionCatalogWrite, http.MethodPut, bomItemPath, func(w http.ResponseWriter, r *http.Request) {
		body, ok := decodeCatalogBOMRequest(w, r)
		if !ok {
			return
		}
		item, err := service.Update(r.Context(), chi.URLParam(r, "itemID"), chi.URLParam(r, "bomItemID"), body.values())
		if err != nil {
			writeCatalogBOMError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, newCatalogBOMItemResponse(item))
	})
	registerCatalogBOMRoute(router, authentication, domainauth.PermissionCatalogWrite, http.MethodDelete, bomItemPath, func(w http.ResponseWriter, r *http.Request) {
		if err := service.Delete(r.Context(), chi.URLParam(r, "itemID"), chi.URLParam(r, "bomItemID")); err != nil {
			writeCatalogBOMError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func (request catalogBOMRequest) values() domaincatalog.BOMValues {
	return domaincatalog.BOMValues{SupplyID: request.SupplyID, QuantityPerUnit: request.QuantityPerUnit, WastePercent: request.WastePercent, Notes: request.Notes}
}

func decodeCatalogBOMRequest(w http.ResponseWriter, r *http.Request) (catalogBOMRequest, bool) {
	if r.Header.Get("Content-Type") != "application/json" {
		WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json", nil)
		return catalogBOMRequest{}, false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, catalogBodyLimit))
	decoder.DisallowUnknownFields()
	var body catalogBOMRequest
	if err := decoder.Decode(&body); err != nil {
		writeInvalidCatalogBOMItem(w)
		return catalogBOMRequest{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeInvalidCatalogBOMItem(w)
		return catalogBOMRequest{}, false
	}
	return body, true
}

func registerCatalogBOMRoute(router *APIV1Router, authentication BearerAuthenticationService, permission domainauth.Permission, method, path string, handler http.HandlerFunc) {
	router.Handle(method, path, RequirePermission(authentication, permission)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		handler(w, r)
	})))
}

func writeCatalogBOMError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, applicationcatalog.ErrInvalidBOMItem):
		writeInvalidCatalogBOMItem(w)
	case errors.Is(err, domaincatalog.ErrItemNotFound):
		writeCatalogNotFound(w)
	case errors.Is(err, domaincatalog.ErrBOMItemNotFound):
		WriteError(w, http.StatusNotFound, "catalog_bom_item_not_found", "Catalog BOM item not found", nil)
	case errors.Is(err, domaininventory.ErrSupplyNotFound):
		WriteError(w, http.StatusNotFound, "supply_not_found", "Supply not found", nil)
	case errors.Is(err, domaincatalog.ErrBOMSupplyConflict):
		WriteError(w, http.StatusConflict, "catalog_bom_supply_exists", "Supply already exists in catalog BOM", nil)
	default:
		WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
	}
}

func writeInvalidCatalogBOMItem(w http.ResponseWriter) {
	WriteError(w, http.StatusBadRequest, "invalid_catalog_bom_item", "Invalid catalog BOM item", nil)
}

func newCatalogBOMItemResponse(item domaincatalog.BOMItem) catalogBOMItemResponse {
	return catalogBOMItemResponse{
		ID: item.ID, CatalogItemID: item.CatalogItemID, SupplyID: item.SupplyID,
		QuantityPerUnit: item.QuantityPerUnit, WastePercent: item.WastePercent,
		Notes: item.Notes, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func newCatalogBOMPreviewResponse(preview domaincatalog.BOMPreview) catalogBOMPreviewResponse {
	items := make([]catalogBOMPreviewLineResponse, 0, len(preview.Items))
	for _, line := range preview.Items {
		items = append(items, catalogBOMPreviewLineResponse{
			catalogBOMItemResponse:           newCatalogBOMItemResponse(line.Item),
			SupplyName:                       line.SupplyName,
			SupplyUnit:                       line.SupplyUnit,
			ReplacementUnitCostCents:         line.ReplacementUnitCostCents,
			EffectiveQuantityPerUnit:         line.EffectiveQuantityPerUnit,
			ExactReplacementCostCentsPerUnit: line.ExactReplacementCostCentsPerUnit,
		})
	}
	return catalogBOMPreviewResponse{Items: items, ExactTotalReplacementCostCents: preview.ExactTotalReplacementCostCents, RoundingApplied: preview.RoundingApplied}
}
