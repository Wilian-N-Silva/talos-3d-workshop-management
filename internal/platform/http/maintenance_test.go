package httpplatform

import (
	"context"
	application "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/maintenance"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/maintenance"
	"net/http"
	"testing"
	"time"
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

func TestMaintenanceInvalidStorageValuesReturnBadRequest(t *testing.T) {
	for _, fields := range []string{`"printer_hours":"."`, `"printer_hours":""`, `"printer_hours":"1.0000"`, `"downtime_minutes":2147483648`} {
		t.Run(fields, func(t *testing.T) {
			service, err := application.NewService(&maintenanceNoWriteRepository{t: t})
			if err != nil {
				t.Fatal(err)
			}
			router := NewAPIV1Router()
			authentication := authorizedCatalogUser(domainauth.RoleOwner)
			authentication.result.User.ID = "22222222-2222-4222-8222-222222222222"
			RegisterMaintenance(router, authentication, service)
			response := inventoryRequest(router, http.MethodPost, PrintersPath+"/11111111-1111-4111-8111-111111111111/maintenance", `{"type":"inspection","performed_at":"2026-09-04T12:00:00Z","description":"Inspect",`+fields+`}`)
			assertAPIError(t, response, http.StatusBadRequest, "invalid_maintenance_event", "Invalid maintenance event")
		})
	}
}

type maintenanceNoWriteRepository struct{ t *testing.T }

func (repository *maintenanceNoWriteRepository) Create(context.Context, string, string, domain.Values, time.Time) (domain.Event, error) {
	repository.t.Fatal("invalid input reached maintenance repository")
	return domain.Event{}, nil
}
func (*maintenanceNoWriteRepository) List(context.Context, string) ([]domain.Event, error) {
	return nil, nil
}

type maintenanceHTTPStub struct{ events []domain.Event }

func (*maintenanceHTTPStub) Create(context.Context, string, string, domain.Values) (domain.Event, error) {
	return domain.Event{}, nil
}
func (s *maintenanceHTTPStub) List(context.Context, string) ([]domain.Event, error) {
	return s.events, nil
}
