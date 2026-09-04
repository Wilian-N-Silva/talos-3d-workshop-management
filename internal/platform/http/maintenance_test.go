package httpplatform

import (
	"context"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/maintenance"
	"net/http"
	"testing"
)

func TestMaintenanceWriteRequiresSettingsManage(t *testing.T) {
	router := NewAPIV1Router()
	RegisterMaintenance(router, authorizedCatalogUser(domainauth.RoleOperator), &maintenanceHTTPStub{})
	response := inventoryRequest(router, http.MethodPost, PrintersPath+"/printer-id/maintenance", `{}`)
	assertAPIError(t, response, http.StatusForbidden, "forbidden", "Permission denied")
}
func TestMaintenanceHistoryIsReadable(t *testing.T) {
	router := NewAPIV1Router()
	RegisterMaintenance(router, authorizedCatalogUser(domainauth.RoleViewer), &maintenanceHTTPStub{events: []domain.Event{{ID: "event-id", Type: domain.TypeInspection}}})
	response := inventoryRequest(router, http.MethodGet, PrintersPath+"/printer-id/maintenance", "")
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

type maintenanceHTTPStub struct{ events []domain.Event }

func (*maintenanceHTTPStub) Create(context.Context, string, string, domain.Values) (domain.Event, error) {
	return domain.Event{}, nil
}
func (s *maintenanceHTTPStub) List(context.Context, string) ([]domain.Event, error) {
	return s.events, nil
}
