package httpplatform

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/printers"
)

const validPrinterBody = `{"name":"A1 Mini","manufacturer":"Bambu Lab","model":"A1 Mini","nozzle_diameter":"0.400","location":"Print room","acquisition_cost_cents":180000,"residual_value_cents":30000,"useful_life_hours":"5000.00","maintenance_reserve_per_hour_cents":25,"status":"active","notes":"Primary"}`

func TestPrinterRoutesExposeLogicalDataAndEnforcePermissions(t *testing.T) {
	now := time.Date(2026, 9, 4, 20, 0, 0, 0, time.UTC)
	printer := domain.Printer{ID: "printer-id", Name: "A1 Mini", Manufacturer: "Bambu Lab", Model: "A1 Mini", NozzleDiameter: "0.4", UsefulLifeHours: "5000", Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now}
	service := &printerServiceStub{printer: printer}
	owner := NewAPIV1Router()
	RegisterPrinters(owner, authorizedCatalogUser(domainauth.RoleOwner), service)
	created := inventoryRequest(owner, http.MethodPost, PrintersPath, validPrinterBody)
	if created.Code != http.StatusCreated || service.values.NozzleDiameter != "0.400" || strings.Contains(created.Body.String(), "access_code") {
		t.Fatalf("create=%d values=%#v body=%s", created.Code, service.values, created.Body.String())
	}

	viewer := NewAPIV1Router()
	RegisterPrinters(viewer, authorizedCatalogUser(domainauth.RoleViewer), service)
	listed := inventoryRequest(viewer, http.MethodGet, PrintersPath, "")
	if listed.Code != http.StatusOK || !strings.Contains(listed.Body.String(), `"nozzle_diameter":"0.4"`) {
		t.Fatalf("list=%d body=%s", listed.Code, listed.Body.String())
	}
	forbidden := inventoryRequest(viewer, http.MethodPost, PrintersPath, validPrinterBody)
	assertAPIError(t, forbidden, http.StatusForbidden, "forbidden", "Permission denied")
}

func TestPrinterRoutesRejectSensitiveAndInvalidFields(t *testing.T) {
	service := &printerServiceStub{}
	router := NewAPIV1Router()
	RegisterPrinters(router, authorizedCatalogUser(domainauth.RoleOwner), service)
	sensitive := strings.TrimSuffix(validPrinterBody, "}") + `,"access_code":"secret"}`
	response := inventoryRequest(router, http.MethodPost, PrintersPath, sensitive)
	assertAPIError(t, response, http.StatusBadRequest, "invalid_printer", "Invalid printer")
}

type printerServiceStub struct {
	printer domain.Printer
	values  domain.Values
	err     error
}

func (stub *printerServiceStub) Create(_ context.Context, values domain.Values) (domain.Printer, error) {
	stub.values = values
	return stub.printer, stub.err
}
func (stub *printerServiceStub) Get(context.Context, string) (domain.Printer, error) {
	return stub.printer, stub.err
}
func (stub *printerServiceStub) List(context.Context) ([]domain.Printer, error) {
	return []domain.Printer{stub.printer}, stub.err
}
func (stub *printerServiceStub) Update(_ context.Context, _ string, values domain.Values) (domain.Printer, error) {
	stub.values = values
	return stub.printer, stub.err
}
func (stub *printerServiceStub) Delete(context.Context, string) error { return stub.err }
