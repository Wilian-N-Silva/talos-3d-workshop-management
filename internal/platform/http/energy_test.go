package httpplatform

import (
	"context"
	"net/http"
	"strings"
	"testing"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/energy"
)

func TestEnergyRoutesRecordActorAndExposeSource(t *testing.T) {
	measured := "1.25"
	service := &energyHTTPStub{measurements: []domain.Measurement{{ID: "measurement-id", JobID: "job-id", Source: domain.SourceManualMeter, MeasuredKWh: &measured, RecordedBy: "user-id"}}}
	router := NewAPIV1Router()
	RegisterEnergy(router, authorizedCatalogUser(domainauth.RoleOperator), service)
	body := `{"source":"manual_meter","meter_start_kwh":"120.125","meter_end_kwh":"121.375","measured_kwh":null,"estimated_average_power_w":null,"energy_rate_cents_per_kwh":95,"occurred_at":"2026-09-04T12:00:00Z","notes":"bench meter"}`
	created := inventoryRequest(router, http.MethodPost, JobsPath+"/job-id/energy", body)
	if created.Code != http.StatusCreated || service.recordedBy == "" || service.values.Source != domain.SourceManualMeter {
		t.Fatalf("create=%d recordedBy=%q values=%#v body=%s", created.Code, service.recordedBy, service.values, created.Body.String())
	}
	listed := inventoryRequest(router, http.MethodGet, JobsPath+"/job-id/energy", "")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"source":"manual_meter"`) || !strings.Contains(listed.Body.String(), `"measured_kwh":"1.25"`) {
		t.Fatalf("list=%d body=%s", listed.Code, listed.Body.String())
	}
}

func TestEnergyWritePermissionIsEnforced(t *testing.T) {
	router := NewAPIV1Router()
	RegisterEnergy(router, authorizedCatalogUser(domainauth.RoleViewer), &energyHTTPStub{})
	response := inventoryRequest(router, http.MethodPost, JobsPath+"/job-id/energy", `{}`)
	assertAPIError(t, response, http.StatusForbidden, "forbidden", "Permission denied")
}

type energyHTTPStub struct {
	values       domain.Values
	recordedBy   string
	measurements []domain.Measurement
}

func (stub *energyHTTPStub) Create(_ context.Context, jobID, recordedBy string, values domain.Values) (domain.Measurement, error) {
	stub.values, stub.recordedBy = values, recordedBy
	return domain.Measurement{ID: "measurement-id", JobID: jobID, Source: values.Source, RecordedBy: recordedBy}, nil
}
func (stub *energyHTTPStub) List(context.Context, string) ([]domain.Measurement, error) {
	return stub.measurements, nil
}
