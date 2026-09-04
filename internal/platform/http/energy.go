package httpplatform

import (
	"context"
	"errors"
	"net/http"
	"time"

	application "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/energy"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/energy"
	domainjobs "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
	"github.com/go-chi/chi/v5"
)

type EnergyService interface {
	Create(context.Context, string, string, domain.Values) (domain.Measurement, error)
	List(context.Context, string) ([]domain.Measurement, error)
}

type energyMeasurementRequest struct {
	Source                 domain.Source `json:"source"`
	MeterStartKWh          *string       `json:"meter_start_kwh"`
	MeterEndKWh            *string       `json:"meter_end_kwh"`
	MeasuredKWh            *string       `json:"measured_kwh"`
	EstimatedAveragePowerW *string       `json:"estimated_average_power_w"`
	EnergyRateCentsPerKWh  int64         `json:"energy_rate_cents_per_kwh"`
	OccurredAt             time.Time     `json:"occurred_at"`
	Notes                  string        `json:"notes"`
}

func RegisterEnergy(router *APIV1Router, authentication BearerAuthenticationService, service EnergyService) {
	path := JobsPath + "/{jobID}/energy"
	registerJobRoute(router, authentication, domainauth.PermissionJobsRead, http.MethodGet, path, func(response http.ResponseWriter, request *http.Request) {
		measurements, err := service.List(request.Context(), chi.URLParam(request, "jobID"))
		if err != nil {
			writeEnergyError(response, err)
			return
		}
		writeJSON(response, http.StatusOK, map[string]any{"measurements": measurements})
	})
	registerJobRoute(router, authentication, domainauth.PermissionJobsUpdate, http.MethodPost, path, func(response http.ResponseWriter, request *http.Request) {
		var body energyMeasurementRequest
		if !decodeJob(response, request, &body) {
			return
		}
		user, ok := CurrentUserFromContext(request.Context())
		if !ok {
			writeUnauthenticated(response)
			return
		}
		measurement, err := service.Create(request.Context(), chi.URLParam(request, "jobID"), user.ID, body.values())
		if err != nil {
			writeEnergyError(response, err)
			return
		}
		writeJSON(response, http.StatusCreated, measurement)
	})
}

func (request energyMeasurementRequest) values() domain.Values {
	return domain.Values{Source: request.Source, MeterStartKWh: request.MeterStartKWh, MeterEndKWh: request.MeterEndKWh, MeasuredKWh: request.MeasuredKWh, EstimatedAveragePowerW: request.EstimatedAveragePowerW, EnergyRateCentsPerKWh: request.EnergyRateCentsPerKWh, OccurredAt: request.OccurredAt, Notes: request.Notes}
}

func writeEnergyError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrInvalidMeasurement):
		WriteError(response, http.StatusBadRequest, "invalid_energy_measurement", "Invalid energy measurement", nil)
	case errors.Is(err, domainjobs.ErrJobNotFound):
		WriteError(response, http.StatusNotFound, "job_not_found", "Print job not found", nil)
	case errors.Is(err, domainjobs.ErrJobReference):
		WriteError(response, http.StatusUnprocessableEntity, "invalid_energy_reference", "Energy measurement reference is invalid", nil)
	default:
		WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
	}
}
