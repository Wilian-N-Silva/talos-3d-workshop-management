package httpplatform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
)

func TestFilamentInventoryRoutesEnforcePermissionsAndExactDecimals(t *testing.T) {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	service := &filamentServiceStub{material: domain.Material{ID: "material-id", Manufacturer: "Maker", Name: "PLA", MaterialType: "PLA", NominalDensity: "1.24", CreatedAt: now, UpdatedAt: now}}
	router := NewAPIV1Router()
	RegisterFilamentInventory(router, authorizedCatalogUser(domainauth.RoleOperator), service)
	body := `{"manufacturer":"Maker","name":"PLA","material_type":"PLA","nominal_density":"1.240000","default_replacement_cost_per_kg_cents":12990}`
	response := inventoryRequest(router, http.MethodPost, MaterialsPath, body)
	if response.Code != http.StatusCreated || service.materialValues.NominalDensity != "1.240000" {
		t.Fatalf("create material=%d values=%#v body=%s", response.Code, service.materialValues, response.Body.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil || decoded["nominal_density"] != "1.24" {
		t.Fatalf("material response=%#v,%v", decoded, err)
	}
	viewer := NewAPIV1Router()
	RegisterFilamentInventory(viewer, authorizedCatalogUser(domainauth.RoleViewer), service)
	forbidden := inventoryRequest(viewer, http.MethodPost, MaterialsPath, body)
	assertAPIError(t, forbidden, http.StatusForbidden, "forbidden", "Permission denied")
}

func TestSpoolMeasurementRoutesRecordActorAndReturnHistory(t *testing.T) {
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	service := &filamentServiceStub{measurement: domain.SpoolMeasurement{ID: "measurement-id", SpoolID: "spool-id", MeasuredAt: now, GrossWeightG: "845.5", DerivedRemainingWeightG: "595.5", Source: domain.MeasurementManual, RecordedBy: "user-id", CreatedAt: now}}
	router := NewAPIV1Router()
	RegisterFilamentInventory(router, authorizedCatalogUser(domainauth.RoleOperator), service)
	response := inventoryRequest(router, http.MethodPost, SpoolsPath+"/spool-id/measurements", `{"measured_at":"2026-09-04T12:00:00Z","gross_weight_g":"845.500","source":"manual","notes":"bench scale"}`)
	if response.Code != http.StatusCreated || service.spoolID != "spool-id" || service.actorID != "user-id" || service.measurementValues.GrossWeightG != "845.500" {
		t.Fatalf("measurement=%d spool/actor=%q/%q values=%#v body=%s", response.Code, service.spoolID, service.actorID, service.measurementValues, response.Body.String())
	}
	history := inventoryRequest(router, http.MethodGet, SpoolsPath+"/spool-id/measurements", "")
	if history.Code != http.StatusOK || !strings.Contains(history.Body.String(), `"derived_remaining_weight_g":"595.5"`) {
		t.Fatalf("history=%d body=%s", history.Code, history.Body.String())
	}
}

func TestInventoryErrorsAreStable(t *testing.T) {
	service := &filamentServiceStub{err: domain.ErrSpoolCodeConflict}
	router := NewAPIV1Router()
	RegisterFilamentInventory(router, authorizedCatalogUser(domainauth.RoleOperator), service)
	response := inventoryRequest(router, http.MethodPost, SpoolsPath, `{"code":"A","material_id":"material","nominal_net_weight_g":"1000","tare_weight_g":"250","status":"sealed"}`)
	assertAPIError(t, response, http.StatusConflict, "spool_code_exists", "Spool code already exists")
}

func inventoryRequest(handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer token")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type filamentServiceStub struct {
	material          domain.Material
	spool             domain.Spool
	measurement       domain.SpoolMeasurement
	materialValues    domain.MaterialValues
	spoolValues       domain.SpoolValues
	measurementValues domain.MeasurementValues
	spoolID           string
	actorID           string
	err               error
}

func (s *filamentServiceStub) CreateMaterial(_ context.Context, v domain.MaterialValues) (domain.Material, error) {
	s.materialValues = v
	return s.material, s.err
}
func (s *filamentServiceStub) GetMaterial(context.Context, string) (domain.Material, error) {
	return s.material, s.err
}
func (s *filamentServiceStub) ListMaterials(context.Context) ([]domain.Material, error) {
	return []domain.Material{s.material}, s.err
}
func (s *filamentServiceStub) UpdateMaterial(_ context.Context, _ string, v domain.MaterialValues) (domain.Material, error) {
	s.materialValues = v
	return s.material, s.err
}
func (s *filamentServiceStub) DeleteMaterial(context.Context, string) error { return s.err }
func (s *filamentServiceStub) CreateSpool(_ context.Context, v domain.SpoolValues) (domain.Spool, error) {
	s.spoolValues = v
	return s.spool, s.err
}
func (s *filamentServiceStub) GetSpool(context.Context, string) (domain.Spool, error) {
	return s.spool, s.err
}
func (s *filamentServiceStub) ListSpools(context.Context) ([]domain.Spool, error) {
	return []domain.Spool{s.spool}, s.err
}
func (s *filamentServiceStub) UpdateSpool(_ context.Context, _ string, v domain.SpoolValues) (domain.Spool, error) {
	s.spoolValues = v
	return s.spool, s.err
}
func (s *filamentServiceStub) DeleteSpool(context.Context, string) error { return s.err }
func (s *filamentServiceStub) RecordMeasurement(_ context.Context, spoolID, actorID string, v domain.MeasurementValues) (domain.SpoolMeasurement, error) {
	s.spoolID, s.actorID, s.measurementValues = spoolID, actorID, v
	return s.measurement, s.err
}
func (s *filamentServiceStub) ListMeasurements(_ context.Context, spoolID string) ([]domain.SpoolMeasurement, error) {
	s.spoolID = spoolID
	return []domain.SpoolMeasurement{s.measurement}, s.err
}
