package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	application "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/printers"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/printers"
	"github.com/go-chi/chi/v5"
)

const (
	PrintersPath     = "/printers"
	printerBodyLimit = 64 * 1024
)

type PrinterService interface {
	Create(context.Context, domain.Values) (domain.Printer, error)
	Get(context.Context, string) (domain.Printer, error)
	List(context.Context) ([]domain.Printer, error)
	Update(context.Context, string, domain.Values) (domain.Printer, error)
	Delete(context.Context, string) error
}

type printerRequest struct {
	Name                           string        `json:"name"`
	Manufacturer                   string        `json:"manufacturer"`
	Model                          string        `json:"model"`
	NozzleDiameter                 string        `json:"nozzle_diameter"`
	Location                       string        `json:"location"`
	AcquisitionCostCents           int64         `json:"acquisition_cost_cents"`
	ResidualValueCents             int64         `json:"residual_value_cents"`
	UsefulLifeHours                string        `json:"useful_life_hours"`
	MaintenanceReservePerHourCents int64         `json:"maintenance_reserve_per_hour_cents"`
	Status                         domain.Status `json:"status"`
	Notes                          string        `json:"notes"`
}

func RegisterPrinters(router *APIV1Router, authentication BearerAuthenticationService, service PrinterService) {
	printerPath := PrintersPath + "/{printerID}"
	registerPrinterRoute(router, authentication, domainauth.PermissionJobsRead, http.MethodGet, PrintersPath, func(w http.ResponseWriter, r *http.Request) {
		printers, err := service.List(r.Context())
		if err != nil {
			writePrinterError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"printers": printers})
	})
	registerPrinterRoute(router, authentication, domainauth.PermissionSettingsManage, http.MethodPost, PrintersPath, func(w http.ResponseWriter, r *http.Request) {
		var body printerRequest
		if !decodePrinter(w, r, &body) {
			return
		}
		printer, err := service.Create(r.Context(), body.values())
		if err != nil {
			writePrinterError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, printer)
	})
	registerPrinterRoute(router, authentication, domainauth.PermissionJobsRead, http.MethodGet, printerPath, func(w http.ResponseWriter, r *http.Request) {
		printer, err := service.Get(r.Context(), chi.URLParam(r, "printerID"))
		if err != nil {
			writePrinterError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, printer)
	})
	registerPrinterRoute(router, authentication, domainauth.PermissionSettingsManage, http.MethodPut, printerPath, func(w http.ResponseWriter, r *http.Request) {
		var body printerRequest
		if !decodePrinter(w, r, &body) {
			return
		}
		printer, err := service.Update(r.Context(), chi.URLParam(r, "printerID"), body.values())
		if err != nil {
			writePrinterError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, printer)
	})
	registerPrinterRoute(router, authentication, domainauth.PermissionSettingsManage, http.MethodDelete, printerPath, func(w http.ResponseWriter, r *http.Request) {
		if err := service.Delete(r.Context(), chi.URLParam(r, "printerID")); err != nil {
			writePrinterError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func (request printerRequest) values() domain.Values {
	return domain.Values{
		Name: request.Name, Manufacturer: request.Manufacturer, Model: request.Model,
		NozzleDiameter: request.NozzleDiameter, Location: request.Location,
		AcquisitionCostCents: request.AcquisitionCostCents, ResidualValueCents: request.ResidualValueCents,
		UsefulLifeHours: request.UsefulLifeHours, MaintenanceReservePerHourCents: request.MaintenanceReservePerHourCents,
		Status: request.Status, Notes: request.Notes,
	}
}

func registerPrinterRoute(router *APIV1Router, authentication BearerAuthenticationService, permission domainauth.Permission, method, path string, handler http.HandlerFunc) {
	router.Handle(method, path, RequirePermission(authentication, permission)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		handler(w, r)
	})))
}

func decodePrinter(w http.ResponseWriter, r *http.Request, target *printerRequest) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json", nil)
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, printerBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeInvalidPrinter(w)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeInvalidPrinter(w)
		return false
	}
	return true
}

func writePrinterError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrInvalidPrinter):
		writeInvalidPrinter(w)
	case errors.Is(err, domain.ErrPrinterNotFound):
		WriteError(w, http.StatusNotFound, "printer_not_found", "Printer not found", nil)
	case errors.Is(err, domain.ErrPrinterNameConflict):
		WriteError(w, http.StatusConflict, "printer_name_exists", "Printer name already exists", nil)
	default:
		WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
	}
}

func writeInvalidPrinter(w http.ResponseWriter) {
	WriteError(w, http.StatusBadRequest, "invalid_printer", "Invalid printer", nil)
}
