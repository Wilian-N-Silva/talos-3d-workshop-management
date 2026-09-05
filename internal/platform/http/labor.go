package httpplatform

import (
	"context"
	"errors"
	"net/http"
	"time"

	application "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/labor"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domainjobs "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/labor"
	"github.com/go-chi/chi/v5"
)

const LaborRatesPath = "/costing/labor-rates"

type LaborService interface {
	CreateRate(context.Context, domain.RateValues) (domain.Rate, error)
	ListRates(context.Context) ([]domain.Rate, error)
	UpdateRate(context.Context, string, domain.RateValues) (domain.Rate, error)
	CreateEntry(context.Context, string, string, domain.EntryValues) (domain.Entry, error)
	ListEntries(context.Context, string) (domain.Summary, error)
}
type laborRateRequest struct {
	Name                string              `json:"name"`
	ActivityType        domain.ActivityType `json:"activity_type"`
	CostHourlyRateCents int64               `json:"cost_hourly_rate_cents"`
	Active              bool                `json:"active"`
}
type laborEntryRequest struct {
	LaborRateID string    `json:"labor_rate_id"`
	Minutes     int       `json:"minutes"`
	OccurredAt  time.Time `json:"occurred_at"`
	Notes       string    `json:"notes"`
}

func RegisterLabor(router *APIV1Router, authentication BearerAuthenticationService, service LaborService) {
	registerLaborRoute(router, authentication, domainauth.PermissionCostingRead, http.MethodGet, LaborRatesPath, func(w http.ResponseWriter, r *http.Request) {
		values, err := service.ListRates(r.Context())
		if err != nil {
			writeLaborError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"rates": values})
	})
	registerLaborRoute(router, authentication, domainauth.PermissionCostingManage, http.MethodPost, LaborRatesPath, func(w http.ResponseWriter, r *http.Request) {
		var body laborRateRequest
		if !decodeJob(w, r, &body) {
			return
		}
		value, err := service.CreateRate(r.Context(), body.values())
		if err != nil {
			writeLaborError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	})
	registerLaborRoute(router, authentication, domainauth.PermissionCostingManage, http.MethodPut, LaborRatesPath+"/{rateID}", func(w http.ResponseWriter, r *http.Request) {
		var body laborRateRequest
		if !decodeJob(w, r, &body) {
			return
		}
		value, err := service.UpdateRate(r.Context(), chi.URLParam(r, "rateID"), body.values())
		if err != nil {
			writeLaborError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	entryPath := JobsPath + "/{jobID}/labor"
	registerLaborRoute(router, authentication, domainauth.PermissionCostingRead, http.MethodGet, entryPath, func(w http.ResponseWriter, r *http.Request) {
		value, err := service.ListEntries(r.Context(), chi.URLParam(r, "jobID"))
		if err != nil {
			writeLaborError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	registerLaborRoute(router, authentication, domainauth.PermissionCostingManage, http.MethodPost, entryPath, func(w http.ResponseWriter, r *http.Request) {
		var body laborEntryRequest
		if !decodeJob(w, r, &body) {
			return
		}
		user, ok := CurrentUserFromContext(r.Context())
		if !ok {
			writeUnauthenticated(w)
			return
		}
		value, err := service.CreateEntry(r.Context(), chi.URLParam(r, "jobID"), user.ID, body.values())
		if err != nil {
			writeLaborError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	})
}
func (request laborRateRequest) values() domain.RateValues {
	return domain.RateValues{Name: request.Name, ActivityType: request.ActivityType, CostHourlyRateCents: request.CostHourlyRateCents, Active: request.Active}
}
func (request laborEntryRequest) values() domain.EntryValues {
	return domain.EntryValues{LaborRateID: request.LaborRateID, Minutes: request.Minutes, OccurredAt: request.OccurredAt, Notes: request.Notes}
}
func registerLaborRoute(router *APIV1Router, authentication BearerAuthenticationService, permission domainauth.Permission, method, path string, handler http.HandlerFunc) {
	router.Handle(method, path, RequirePermission(authentication, permission)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		handler(w, r)
	})))
}
func writeLaborError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrInvalidRate), errors.Is(err, application.ErrInvalidEntry):
		WriteError(w, http.StatusBadRequest, "invalid_labor_data", "Invalid labor data", nil)
	case errors.Is(err, domain.ErrRateNotFound):
		WriteError(w, http.StatusNotFound, "labor_rate_not_found", "Labor rate not found or inactive", nil)
	case errors.Is(err, domain.ErrRateConflict):
		WriteError(w, http.StatusConflict, "labor_rate_exists", "Labor rate name already exists", nil)
	case errors.Is(err, domainjobs.ErrJobNotFound):
		WriteError(w, http.StatusNotFound, "job_not_found", "Print job not found", nil)
	case errors.Is(err, domainjobs.ErrJobReference):
		WriteError(w, http.StatusUnprocessableEntity, "invalid_labor_reference", "Labor entry reference is invalid", nil)
	default:
		WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
	}
}
