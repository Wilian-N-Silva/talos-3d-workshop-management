package httpplatform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/labor"
)

func TestLaborRateAndEntryWritesRequireCostingManage(t *testing.T) {
	for _, path := range []string{LaborRatesPath, JobsPath + "/job-id/labor"} {
		router := NewAPIV1Router()
		RegisterLabor(router, authorizedCatalogUser(domainauth.RoleViewer), &laborHTTPStub{})
		response := inventoryRequest(router, http.MethodPost, path, `{}`)
		assertAPIError(t, response, http.StatusForbidden, "forbidden", "Permission denied")
	}
}
func TestLaborEntryCapturesAuthenticatedRecorder(t *testing.T) {
	service := &laborHTTPStub{}
	router := NewAPIV1Router()
	RegisterLabor(router, authorizedCatalogUser(domainauth.RoleOwner), service)
	body := `{"labor_rate_id":"33333333-3333-4333-8333-333333333333","minutes":15,"occurred_at":"2026-09-04T12:00:00Z","notes":"setup"}`
	response := inventoryRequest(router, http.MethodPost, JobsPath+"/11111111-1111-4111-8111-111111111111/labor", body)
	if response.Code != http.StatusCreated || service.recordedBy == "" || service.entry.Minutes != 15 {
		t.Fatalf("response=%d recorder=%q entry=%#v body=%s", response.Code, service.recordedBy, service.entry, response.Body.String())
	}
}

type laborHTTPStub struct {
	recordedBy string
	entry      domain.EntryValues
}

func (*laborHTTPStub) CreateRate(context.Context, domain.RateValues) (domain.Rate, error) {
	return domain.Rate{}, nil
}
func (*laborHTTPStub) ListRates(context.Context) ([]domain.Rate, error) { return []domain.Rate{}, nil }
func (*laborHTTPStub) UpdateRate(context.Context, string, domain.RateValues) (domain.Rate, error) {
	return domain.Rate{}, nil
}
func (s *laborHTTPStub) CreateEntry(_ context.Context, _ string, recordedBy string, input domain.EntryValues) (domain.Entry, error) {
	s.recordedBy, s.entry = recordedBy, input
	return domain.Entry{}, nil
}
func (*laborHTTPStub) ListEntries(context.Context, string) (domain.Summary, error) {
	return domain.Summary{Items: []domain.Entry{}, MinutesByActivity: map[domain.ActivityType]int64{}}, nil
}

func TestLaborSuggestionExactFormulaAndValidation(t *testing.T) {
	router := NewAPIV1Router()
	RegisterLabor(router, authorizedCatalogUser(domainauth.RoleOwner), &laborHTTPStub{})
	response := inventoryRequest(router, http.MethodPost, LaborRatesPath+"/suggestion", `{"target_monthly_compensation_cents":300000,"monthly_labor_overhead_cents":50000,"available_hours_per_month":"160","productive_utilization_bps":7500}`)
	var result struct {
		Hours string `json:"productive_hours"`
		Cents int64  `json:"internal_hourly_cost_cents"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || response.Code != 200 || result.Hours != "120.0000000000" || result.Cents != 2917 || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	for _, body := range []string{
		`{}`, `{"target_monthly_compensation_cents":0,"available_hours_per_month":"160","productive_utilization_bps":7500}`,
		`{"target_monthly_compensation_cents":0,"monthly_labor_overhead_cents":0,"available_hours_per_month":"0","productive_utilization_bps":7500}`,
		`{"target_monthly_compensation_cents":1.5,"monthly_labor_overhead_cents":0,"available_hours_per_month":"160","productive_utilization_bps":7500}`,
		`{"target_monthly_compensation_cents":9223372036854775807,"monthly_labor_overhead_cents":0,"available_hours_per_month":"0.1","productive_utilization_bps":10000}`,
	} {
		response := inventoryRequest(router, http.MethodPost, LaborRatesPath+"/suggestion", body)
		if response.Code != 400 {
			t.Fatalf("accepted %s: %d", body, response.Code)
		}
	}
}
func TestLaborSuggestionRequiresAuthenticationAndReadPermission(t *testing.T) {
	router := NewAPIV1Router()
	RegisterLabor(router, authorizedCatalogUser(domainauth.RoleOwner), &laborHTTPStub{})
	request := httptest.NewRequest(http.MethodPost, LaborRatesPath+"/suggestion", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 401 {
		t.Fatalf("anonymous status=%d", response.Code)
	}
	router = NewAPIV1Router()
	RegisterLabor(router, authorizedCatalogUser(domainauth.RoleOperator), &laborHTTPStub{})
	response = inventoryRequest(router, http.MethodPost, LaborRatesPath+"/suggestion", `{}`)
	if response.Code != 403 {
		t.Fatalf("operator status=%d", response.Code)
	}
}
