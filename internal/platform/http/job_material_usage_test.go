package httpplatform

import (
	"context"
	"net/http"
	"strings"
	"testing"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
)

const materialUsageBody = `{"spool_id":"66666666-6666-4666-8666-666666666666","role":"model","planned_grams":"12.5","actual_grams":"11.75","measurement_source":"spool_weight_delta"}`

func TestJobMaterialUsageRoutesExposeSourceAndTotals(t *testing.T) {
	actual := "11.75"
	service := &jobMaterialUsageHTTPStub{summary: domain.MaterialUsageSummary{
		Items:             []domain.MaterialUsage{{ID: "usage-id", MeasurementSource: domain.SourceSpoolWeightDelta, PlannedGrams: "12.5", ActualGrams: &actual}},
		TotalPlannedGrams: "12.5",
		TotalActualGrams:  "11.75",
	}}
	router := NewAPIV1Router()
	RegisterJobMaterialUsage(router, authorizedCatalogUser(domainauth.RoleOperator), service)

	created := inventoryRequest(router, http.MethodPost, JobsPath+"/job-id/materials", materialUsageBody)
	if created.Code != http.StatusCreated || service.created.MeasurementSource != domain.SourceSpoolWeightDelta || service.created.ActualGrams == nil || *service.created.ActualGrams != actual {
		t.Fatalf("create=%d values=%#v body=%s", created.Code, service.created, created.Body.String())
	}
	listed := inventoryRequest(router, http.MethodGet, JobsPath+"/job-id/materials", "")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"measurement_source":"spool_weight_delta"`) || !strings.Contains(listed.Body.String(), `"total_planned_grams":"12.5"`) || !strings.Contains(listed.Body.String(), `"total_actual_grams":"11.75"`) {
		t.Fatalf("list=%d body=%s", listed.Code, listed.Body.String())
	}
}

func TestJobMaterialUsageWritePermissionIsEnforced(t *testing.T) {
	router := NewAPIV1Router()
	RegisterJobMaterialUsage(router, authorizedCatalogUser(domainauth.RoleViewer), &jobMaterialUsageHTTPStub{})
	response := inventoryRequest(router, http.MethodPost, JobsPath+"/job-id/materials", materialUsageBody)
	assertAPIError(t, response, http.StatusForbidden, "forbidden", "Permission denied")
}

type jobMaterialUsageHTTPStub struct {
	created domain.MaterialUsageValues
	summary domain.MaterialUsageSummary
}

func (s *jobMaterialUsageHTTPStub) Create(_ context.Context, _ string, values domain.MaterialUsageValues) (domain.MaterialUsage, error) {
	s.created = values
	return domain.MaterialUsage{ID: "usage-id", MeasurementSource: values.MeasurementSource, PlannedGrams: values.PlannedGrams, ActualGrams: values.ActualGrams}, nil
}
func (s *jobMaterialUsageHTTPStub) List(context.Context, string) (domain.MaterialUsageSummary, error) {
	return s.summary, nil
}
func (s *jobMaterialUsageHTTPStub) Update(_ context.Context, _, _ string, values domain.MaterialUsageValues) (domain.MaterialUsage, error) {
	return domain.MaterialUsage{ID: "usage-id", MeasurementSource: values.MeasurementSource}, nil
}
func (s *jobMaterialUsageHTTPStub) Delete(context.Context, string, string) error { return nil }
