package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	application "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/inventory"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
	"github.com/go-chi/chi/v5"
)

const (
	MaterialsPath      = "/inventory/materials"
	SpoolsPath         = "/inventory/spools"
	inventoryBodyLimit = 64 * 1024
)

type FilamentInventoryService interface {
	CreateMaterial(context.Context, domain.MaterialValues) (domain.Material, error)
	GetMaterial(context.Context, string) (domain.Material, error)
	ListMaterials(context.Context) ([]domain.Material, error)
	UpdateMaterial(context.Context, string, domain.MaterialValues) (domain.Material, error)
	DeleteMaterial(context.Context, string) error
	CreateSpool(context.Context, domain.SpoolValues) (domain.Spool, error)
	GetSpool(context.Context, string) (domain.Spool, error)
	ListSpools(context.Context) ([]domain.Spool, error)
	UpdateSpool(context.Context, string, domain.SpoolValues) (domain.Spool, error)
	DeleteSpool(context.Context, string) error
	RecordMeasurement(context.Context, string, string, domain.MeasurementValues) (domain.SpoolMeasurement, error)
	ListMeasurements(context.Context, string) ([]domain.SpoolMeasurement, error)
}

type materialRequest struct {
	Manufacturer                     string  `json:"manufacturer"`
	Name                             string  `json:"name"`
	MaterialType                     string  `json:"material_type"`
	ColorName                        string  `json:"color_name"`
	ColorHex                         *string `json:"color_hex"`
	NominalDensity                   string  `json:"nominal_density"`
	DefaultReplacementCostPerKgCents int64   `json:"default_replacement_cost_per_kg_cents"`
	Notes                            string  `json:"notes"`
}
type spoolRequest struct {
	Code                      string             `json:"code"`
	MaterialID                string             `json:"material_id"`
	NominalNetWeightG         string             `json:"nominal_net_weight_g"`
	TareWeightG               string             `json:"tare_weight_g"`
	GrossWeightAtOpenG        *string            `json:"gross_weight_at_open_g"`
	PurchaseCostCents         int64              `json:"purchase_cost_cents"`
	ReplacementCostPerKgCents int64              `json:"replacement_cost_per_kg_cents"`
	OpenedAt                  *time.Time         `json:"opened_at"`
	LastDriedAt               *time.Time         `json:"last_dried_at"`
	StorageLocation           string             `json:"storage_location"`
	StorageStatus             string             `json:"storage_status"`
	LotNumber                 string             `json:"lot_number"`
	Status                    domain.SpoolStatus `json:"status"`
}
type measurementRequest struct {
	MeasuredAt   time.Time                `json:"measured_at"`
	GrossWeightG string                   `json:"gross_weight_g"`
	Source       domain.MeasurementSource `json:"source"`
	Notes        string                   `json:"notes"`
}

func RegisterFilamentInventory(router *APIV1Router, authentication BearerAuthenticationService, service FilamentInventoryService) {
	materialPath := MaterialsPath + "/{materialID}"
	spoolPath := SpoolsPath + "/{spoolID}"
	measurementsPath := spoolPath + "/measurements"
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryRead, http.MethodGet, MaterialsPath, func(w http.ResponseWriter, r *http.Request) {
		values, err := service.ListMaterials(r.Context())
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"materials": values})
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryWrite, http.MethodPost, MaterialsPath, func(w http.ResponseWriter, r *http.Request) {
		var body materialRequest
		if !decodeInventory(w, r, &body) {
			return
		}
		value, err := service.CreateMaterial(r.Context(), body.values())
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryRead, http.MethodGet, materialPath, func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetMaterial(r.Context(), chi.URLParam(r, "materialID"))
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryWrite, http.MethodPut, materialPath, func(w http.ResponseWriter, r *http.Request) {
		var body materialRequest
		if !decodeInventory(w, r, &body) {
			return
		}
		value, err := service.UpdateMaterial(r.Context(), chi.URLParam(r, "materialID"), body.values())
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryWrite, http.MethodDelete, materialPath, func(w http.ResponseWriter, r *http.Request) {
		if err := service.DeleteMaterial(r.Context(), chi.URLParam(r, "materialID")); err != nil {
			writeInventoryError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryRead, http.MethodGet, SpoolsPath, func(w http.ResponseWriter, r *http.Request) {
		values, err := service.ListSpools(r.Context())
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"spools": values})
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryWrite, http.MethodPost, SpoolsPath, func(w http.ResponseWriter, r *http.Request) {
		var body spoolRequest
		if !decodeInventory(w, r, &body) {
			return
		}
		value, err := service.CreateSpool(r.Context(), body.values())
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryRead, http.MethodGet, spoolPath, func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetSpool(r.Context(), chi.URLParam(r, "spoolID"))
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryWrite, http.MethodPut, spoolPath, func(w http.ResponseWriter, r *http.Request) {
		var body spoolRequest
		if !decodeInventory(w, r, &body) {
			return
		}
		value, err := service.UpdateSpool(r.Context(), chi.URLParam(r, "spoolID"), body.values())
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryWrite, http.MethodDelete, spoolPath, func(w http.ResponseWriter, r *http.Request) {
		if err := service.DeleteSpool(r.Context(), chi.URLParam(r, "spoolID")); err != nil {
			writeInventoryError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryRead, http.MethodGet, measurementsPath, func(w http.ResponseWriter, r *http.Request) {
		values, err := service.ListMeasurements(r.Context(), chi.URLParam(r, "spoolID"))
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"measurements": values})
	})
	registerInventoryRoute(router, authentication, domainauth.PermissionInventoryWrite, http.MethodPost, measurementsPath, func(w http.ResponseWriter, r *http.Request) {
		var body measurementRequest
		if !decodeInventory(w, r, &body) {
			return
		}
		user, ok := CurrentUserFromContext(r.Context())
		if !ok {
			writeUnauthenticated(w)
			return
		}
		value, err := service.RecordMeasurement(r.Context(), chi.URLParam(r, "spoolID"), user.ID, domain.MeasurementValues{MeasuredAt: body.MeasuredAt, GrossWeightG: body.GrossWeightG, Source: body.Source, Notes: body.Notes})
		if err != nil {
			writeInventoryError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	})
}

func (request materialRequest) values() domain.MaterialValues {
	return domain.MaterialValues{Manufacturer: request.Manufacturer, Name: request.Name, MaterialType: request.MaterialType, ColorName: request.ColorName, ColorHex: request.ColorHex, NominalDensity: request.NominalDensity, DefaultReplacementCostPerKgCents: request.DefaultReplacementCostPerKgCents, Notes: request.Notes}
}
func (request spoolRequest) values() domain.SpoolValues {
	return domain.SpoolValues{Code: request.Code, MaterialID: request.MaterialID, NominalNetWeightG: request.NominalNetWeightG, TareWeightG: request.TareWeightG, GrossWeightAtOpenG: request.GrossWeightAtOpenG, PurchaseCostCents: request.PurchaseCostCents, ReplacementCostPerKgCents: request.ReplacementCostPerKgCents, OpenedAt: request.OpenedAt, LastDriedAt: request.LastDriedAt, StorageLocation: request.StorageLocation, StorageStatus: request.StorageStatus, LotNumber: request.LotNumber, Status: request.Status}
}
func registerInventoryRoute(router *APIV1Router, authentication BearerAuthenticationService, permission domainauth.Permission, method, path string, handler http.HandlerFunc) {
	router.Handle(method, path, RequirePermission(authentication, permission)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		handler(w, r)
	})))
}
func decodeInventory(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json", nil)
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, inventoryBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_inventory_data", "Invalid inventory data", nil)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		WriteError(w, http.StatusBadRequest, "invalid_inventory_data", "Invalid inventory data", nil)
		return false
	}
	return true
}
func writeInventoryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrInvalidMaterial), errors.Is(err, application.ErrInvalidSpool), errors.Is(err, application.ErrInvalidMeasurement), errors.Is(err, application.ErrInvalidSupply), errors.Is(err, application.ErrInvalidSupplyMovement), errors.Is(err, application.ErrInvalidLowThreshold), errors.Is(err, domain.ErrMeasurementBelowTare):
		WriteError(w, http.StatusBadRequest, "invalid_inventory_data", "Invalid inventory data", nil)
	case errors.Is(err, domain.ErrMaterialNotFound):
		WriteError(w, http.StatusNotFound, "material_not_found", "Material not found", nil)
	case errors.Is(err, domain.ErrSpoolNotFound):
		WriteError(w, http.StatusNotFound, "spool_not_found", "Spool not found", nil)
	case errors.Is(err, domain.ErrSpoolCodeConflict):
		WriteError(w, http.StatusConflict, "spool_code_exists", "Spool code already exists", nil)
	case errors.Is(err, domain.ErrInventoryHistoryExists):
		WriteError(w, http.StatusConflict, "inventory_history_exists", "Inventory history prevents deletion", nil)
	case errors.Is(err, domain.ErrSupplyNotFound):
		WriteError(w, http.StatusNotFound, "supply_not_found", "Supply not found", nil)
	case errors.Is(err, domain.ErrSupplySKUConflict):
		WriteError(w, http.StatusConflict, "supply_sku_exists", "Supply SKU already exists", nil)
	case errors.Is(err, domain.ErrSupplyHistoryExists):
		WriteError(w, http.StatusConflict, "supply_history_exists", "Supply movement history prevents deletion", nil)
	case errors.Is(err, domain.ErrInsufficientStock):
		WriteError(w, http.StatusConflict, "insufficient_stock", "Movement would make supply stock negative", nil)
	default:
		WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
	}
}
