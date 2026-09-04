package httpplatform

import (
	"context"
	"errors"
	application "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/jobs"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domaininventory "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
	"github.com/go-chi/chi/v5"
	"net/http"
)

type JobMaterialUsageService interface {
	Create(context.Context, string, domain.MaterialUsageValues) (domain.MaterialUsage, error)
	List(context.Context, string) (domain.MaterialUsageSummary, error)
	Update(context.Context, string, string, domain.MaterialUsageValues) (domain.MaterialUsage, error)
	Delete(context.Context, string, string) error
}
type materialUsageRequest struct {
	SpoolID                      string                   `json:"spool_id"`
	Role                         domain.MaterialRole      `json:"role"`
	PlannedGrams                 string                   `json:"planned_grams"`
	ActualGrams                  *string                  `json:"actual_grams"`
	PlannedMeters                *string                  `json:"planned_meters"`
	ActualMeters                 *string                  `json:"actual_meters"`
	MeasurementSource            domain.MeasurementSource `json:"measurement_source"`
	HistoricalMaterialCostCents  *int64                   `json:"historical_material_cost_cents"`
	ReplacementMaterialCostCents *int64                   `json:"replacement_material_cost_cents"`
}

func RegisterJobMaterialUsage(router *APIV1Router, auth BearerAuthenticationService, service JobMaterialUsageService) {
	path := JobsPath + "/{jobID}/materials"
	itemPath := path + "/{usageID}"
	registerJobRoute(router, auth, domainauth.PermissionJobsRead, http.MethodGet, path, func(w http.ResponseWriter, r *http.Request) {
		summary, err := service.List(r.Context(), chi.URLParam(r, "jobID"))
		if err != nil {
			writeMaterialUsageError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, summary)
	})
	registerJobRoute(router, auth, domainauth.PermissionJobsUpdate, http.MethodPost, path, func(w http.ResponseWriter, r *http.Request) {
		var body materialUsageRequest
		if !decodeJob(w, r, &body) {
			return
		}
		value, err := service.Create(r.Context(), chi.URLParam(r, "jobID"), body.values())
		if err != nil {
			writeMaterialUsageError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	})
	registerJobRoute(router, auth, domainauth.PermissionJobsUpdate, http.MethodPut, itemPath, func(w http.ResponseWriter, r *http.Request) {
		var body materialUsageRequest
		if !decodeJob(w, r, &body) {
			return
		}
		value, err := service.Update(r.Context(), chi.URLParam(r, "jobID"), chi.URLParam(r, "usageID"), body.values())
		if err != nil {
			writeMaterialUsageError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	registerJobRoute(router, auth, domainauth.PermissionJobsUpdate, http.MethodDelete, itemPath, func(w http.ResponseWriter, r *http.Request) {
		if err := service.Delete(r.Context(), chi.URLParam(r, "jobID"), chi.URLParam(r, "usageID")); err != nil {
			writeMaterialUsageError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
func (r materialUsageRequest) values() domain.MaterialUsageValues {
	return domain.MaterialUsageValues{SpoolID: r.SpoolID, Role: r.Role, PlannedGrams: r.PlannedGrams, ActualGrams: r.ActualGrams, PlannedMeters: r.PlannedMeters, ActualMeters: r.ActualMeters, MeasurementSource: r.MeasurementSource, HistoricalMaterialCostCents: r.HistoricalMaterialCostCents, ReplacementMaterialCostCents: r.ReplacementMaterialCostCents}
}
func writeMaterialUsageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrInvalidMaterialUsage):
		WriteError(w, http.StatusBadRequest, "invalid_job_material_usage", "Invalid job material usage", nil)
	case errors.Is(err, domain.ErrJobNotFound):
		WriteError(w, http.StatusNotFound, "job_not_found", "Print job not found", nil)
	case errors.Is(err, domain.ErrMaterialUsageNotFound):
		WriteError(w, http.StatusNotFound, "job_material_usage_not_found", "Job material usage not found", nil)
	case errors.Is(err, domaininventory.ErrSpoolNotFound):
		WriteError(w, http.StatusNotFound, "spool_not_found", "Spool not found", nil)
	case errors.Is(err, domain.ErrMaterialUsageConflict):
		WriteError(w, http.StatusConflict, "job_material_usage_exists", "Spool and role already exist on this Job", nil)
	case errors.Is(err, domain.ErrJobNotEditable):
		WriteError(w, http.StatusConflict, "invalid_job_state", "Print job state does not allow this operation", nil)
	default:
		WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
	}
}
