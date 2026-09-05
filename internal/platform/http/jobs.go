package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	application "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/jobs"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
	"github.com/go-chi/chi/v5"
)

const JobsPath = "/jobs"
const jobBodyLimit = 64 * 1024

type JobService interface {
	Create(context.Context, domain.Values, domain.Actor) (domain.Job, error)
	Get(context.Context, string) (domain.Job, error)
	List(context.Context) ([]domain.Job, error)
	Update(context.Context, string, domain.Values) (domain.Job, error)
	Transition(context.Context, string, domain.TransitionValues, domain.Actor) (domain.Job, error)
	Review(context.Context, string, domain.ReviewValues, domain.Actor) (domain.Job, error)
	Delete(context.Context, string) error
	ListEvents(context.Context, string) ([]domain.Event, error)
}

type jobRequest struct {
	Code            string         `json:"code"`
	CatalogItemID   string         `json:"catalog_item_id"`
	DesignVersionID string         `json:"design_version_id"`
	PrinterID       string         `json:"printer_id"`
	Purpose         domain.Purpose `json:"purpose"`
	PlannedQuantity int            `json:"planned_quantity"`
	Hypothesis      string         `json:"hypothesis"`
	PlannedSeconds  int64          `json:"planned_seconds"`
	LaborMinutes    int            `json:"labor_minutes"`
}
type jobTransitionRequest struct {
	Status        domain.Status `json:"status"`
	ActualSeconds *int64        `json:"actual_seconds"`
	ResultNotes   string        `json:"result_notes"`
}
type jobReviewRequest struct {
	QualityStatus domain.QualityStatus `json:"quality_status"`
	GoodQuantity  int                  `json:"good_quantity"`
	ScrapQuantity int                  `json:"scrap_quantity"`
	ResultNotes   string               `json:"result_notes"`
}

func RegisterJobs(router *APIV1Router, authentication BearerAuthenticationService, service JobService) {
	jobPath := JobsPath + "/{jobID}"
	transitionPath := jobPath + "/transitions"
	reviewPath := jobPath + "/review"
	eventsPath := jobPath + "/events"
	registerJobRoute(router, authentication, domainauth.PermissionJobsRead, http.MethodGet, JobsPath, func(w http.ResponseWriter, r *http.Request) {
		values, err := service.List(r.Context())
		if err != nil {
			writeJobError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"jobs": values})
	})
	registerJobRoute(router, authentication, domainauth.PermissionJobsCreate, http.MethodPost, JobsPath, func(w http.ResponseWriter, r *http.Request) {
		var body jobRequest
		if !decodeJob(w, r, &body) {
			return
		}
		actor, ok := jobActor(r.Context())
		if !ok {
			writeUnauthenticated(w)
			return
		}
		value, err := service.Create(r.Context(), body.values(), actor)
		if err != nil {
			writeJobError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, value)
	})
	registerJobRoute(router, authentication, domainauth.PermissionJobsRead, http.MethodGet, jobPath, func(w http.ResponseWriter, r *http.Request) {
		value, err := service.Get(r.Context(), chi.URLParam(r, "jobID"))
		if err != nil {
			writeJobError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	registerJobRoute(router, authentication, domainauth.PermissionJobsUpdate, http.MethodPut, jobPath, func(w http.ResponseWriter, r *http.Request) {
		var body jobRequest
		if !decodeJob(w, r, &body) {
			return
		}
		value, err := service.Update(r.Context(), chi.URLParam(r, "jobID"), body.values())
		if err != nil {
			writeJobError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	registerJobRoute(router, authentication, domainauth.PermissionJobsUpdate, http.MethodDelete, jobPath, func(w http.ResponseWriter, r *http.Request) {
		if err := service.Delete(r.Context(), chi.URLParam(r, "jobID")); err != nil {
			writeJobError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	registerJobRoute(router, authentication, domainauth.PermissionJobsUpdate, http.MethodPost, transitionPath, func(w http.ResponseWriter, r *http.Request) {
		var body jobTransitionRequest
		if !decodeJob(w, r, &body) {
			return
		}
		actor, ok := jobActor(r.Context())
		if !ok {
			writeUnauthenticated(w)
			return
		}
		value, err := service.Transition(r.Context(), chi.URLParam(r, "jobID"), domain.TransitionValues{Status: body.Status, ActualSeconds: body.ActualSeconds, ResultNotes: body.ResultNotes}, actor)
		if err != nil {
			writeJobError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	registerJobRoute(router, authentication, domainauth.PermissionJobsEvaluate, http.MethodPost, reviewPath, func(w http.ResponseWriter, r *http.Request) {
		var body jobReviewRequest
		if !decodeJob(w, r, &body) {
			return
		}
		actor, ok := jobActor(r.Context())
		if !ok {
			writeUnauthenticated(w)
			return
		}
		value, err := service.Review(r.Context(), chi.URLParam(r, "jobID"), domain.ReviewValues{QualityStatus: body.QualityStatus, GoodQuantity: body.GoodQuantity, ScrapQuantity: body.ScrapQuantity, ResultNotes: body.ResultNotes}, actor)
		if err != nil {
			writeJobError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, value)
	})
	registerJobRoute(router, authentication, domainauth.PermissionJobsRead, http.MethodGet, eventsPath, func(w http.ResponseWriter, r *http.Request) {
		values, err := service.ListEvents(r.Context(), chi.URLParam(r, "jobID"))
		if err != nil {
			writeJobError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": values})
	})
}
func (r jobRequest) values() domain.Values {
	return domain.Values{Code: r.Code, CatalogItemID: r.CatalogItemID, DesignVersionID: r.DesignVersionID, PrinterID: r.PrinterID, Purpose: r.Purpose, PlannedQuantity: r.PlannedQuantity, Hypothesis: r.Hypothesis, PlannedSeconds: r.PlannedSeconds, LaborMinutes: r.LaborMinutes}
}
func jobActor(ctx context.Context) (domain.Actor, bool) {
	user, uok := CurrentUserFromContext(ctx)
	session, sok := CurrentSessionFromContext(ctx)
	return domain.Actor{UserID: user.ID, DeviceID: session.DeviceID}, uok && sok
}
func registerJobRoute(router *APIV1Router, auth BearerAuthenticationService, p domainauth.Permission, method, path string, handler http.HandlerFunc) {
	router.Handle(method, path, RequirePermission(auth, p)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		handler(w, r)
	})))
}
func decodeJob(w http.ResponseWriter, r *http.Request, target any) bool {
	if r.Header.Get("Content-Type") != "application/json" {
		WriteError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json", nil)
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, jobBodyLimit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeInvalidJob(w)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeInvalidJob(w)
		return false
	}
	return true
}
func writeJobError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrInvalidJob), errors.Is(err, application.ErrInvalidTransition), errors.Is(err, application.ErrInvalidReview):
		writeInvalidJob(w)
	case errors.Is(err, domain.ErrJobNotFound):
		WriteError(w, http.StatusNotFound, "job_not_found", "Print job not found", nil)
	case errors.Is(err, domain.ErrJobCodeConflict):
		WriteError(w, http.StatusConflict, "job_code_exists", "Print job code already exists", nil)
	case errors.Is(err, domain.ErrJobReference):
		WriteError(w, http.StatusUnprocessableEntity, "invalid_job_reference", "Catalog, design, printer, user, or device reference is invalid", nil)
	case errors.Is(err, domain.ErrJobStateConflict), errors.Is(err, domain.ErrJobNotEditable), errors.Is(err, domain.ErrJobNotDeletable):
		WriteError(w, http.StatusConflict, "invalid_job_state", "Print job state does not allow this operation", nil)
	default:
		WriteError(w, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
	}
}
func writeInvalidJob(w http.ResponseWriter) {
	WriteError(w, http.StatusBadRequest, "invalid_job", "Invalid print job data", nil)
}
