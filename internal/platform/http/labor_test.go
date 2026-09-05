package httpplatform

import (
	"context"
	"net/http"
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
