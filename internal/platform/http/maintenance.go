package httpplatform

import (
	"context"
	"errors"
	application "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/maintenance"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/maintenance"
	domainprinters "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/printers"
	"github.com/go-chi/chi/v5"
	"net/http"
	"time"
)

type MaintenanceService interface {
	Create(context.Context, string, string, domain.Values) (domain.Event, error)
	List(context.Context, string) ([]domain.Event, error)
}
type maintenanceRequest struct {
	Type            domain.Type `json:"type"`
	PerformedAt     time.Time   `json:"performed_at"`
	PrinterHours    *string     `json:"printer_hours"`
	Description     string      `json:"description"`
	CostCents       *int64      `json:"cost_cents"`
	DowntimeMinutes int         `json:"downtime_minutes"`
	Notes           string      `json:"notes"`
}

func RegisterMaintenance(router *APIV1Router, authentication BearerAuthenticationService, service MaintenanceService) {
	path := PrintersPath + "/{printerID}/maintenance"
	registerJobRoute(router, authentication, domainauth.PermissionJobsRead, http.MethodGet, path, func(w http.ResponseWriter, r *http.Request) {
		events, err := service.List(r.Context(), chi.URLParam(r, "printerID"))
		if err != nil {
			writeMaintenanceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": events})
	})
	registerJobRoute(router, authentication, domainauth.PermissionSettingsManage, http.MethodPost, path, func(w http.ResponseWriter, r *http.Request) {
		var body maintenanceRequest
		if !decodeJob(w, r, &body) {
			return
		}
		user, ok := CurrentUserFromContext(r.Context())
		if !ok {
			writeUnauthenticated(w)
			return
		}
		event, err := service.Create(r.Context(), chi.URLParam(r, "printerID"), user.ID, body.values())
		if err != nil {
			writeMaintenanceError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, event)
	})
}
func (request maintenanceRequest) values() domain.Values {
	return domain.Values{Type: request.Type, PerformedAt: request.PerformedAt, PrinterHours: request.PrinterHours, Description: request.Description, CostCents: request.CostCents, DowntimeMinutes: request.DowntimeMinutes, Notes: request.Notes}
}
func writeMaintenanceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrInvalidEvent):
		WriteError(w, http.StatusBadRequest, "invalid_maintenance_event", "Invalid maintenance event", nil)
	case errors.Is(err, domainprinters.ErrPrinterNotFound):
		WriteError(w, http.StatusNotFound, "printer_not_found", "Printer not found", nil)
	case errors.Is(err, domain.ErrMaintenanceReference):
		WriteError(w, http.StatusUnprocessableEntity, "invalid_maintenance_reference", "Maintenance event reference is invalid", nil)
	default:
		WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
	}
}
