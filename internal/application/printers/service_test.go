package printers

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/printers"
)

const printerID = "11111111-1111-4111-8111-111111111111"

func TestServiceNormalizesAndPersistsExactPrinterValues(t *testing.T) {
	repository := &repositoryStub{}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService()=%v", err)
	}
	now := time.Date(2026, 9, 4, 20, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	printer, err := service.Create(context.Background(), domain.Values{Name: " A1 Mini ", Manufacturer: " Bambu Lab ", Model: " A1 Mini ", NozzleDiameter: "0.400", Location: " Bench ", AcquisitionCostCents: 180000, ResidualValueCents: 30000, UsefulLifeHours: "5000.00", MaintenanceReservePerHourCents: 25, Notes: " Primary "})
	if err != nil || printer.Name != "A1 Mini" || repository.values.NozzleDiameter != "0.4" || repository.values.UsefulLifeHours != "5000" || repository.values.Status != domain.StatusActive || !repository.at.Equal(now) {
		t.Fatalf("Create()=%#v,%v values=%#v at=%v", printer, err, repository.values, repository.at)
	}
}

func TestServiceRejectsInvalidFinancialAndPhysicalValues(t *testing.T) {
	service, _ := NewService(&repositoryStub{})
	base := domain.Values{Name: "A1", Manufacturer: "Bambu", Model: "A1", NozzleDiameter: "0.4", AcquisitionCostCents: 100, ResidualValueCents: 10, UsefulLifeHours: "1000", Status: domain.StatusActive}
	tests := []domain.Values{
		func() domain.Values { value := base; value.NozzleDiameter = "0"; return value }(),
		func() domain.Values { value := base; value.UsefulLifeHours = "0"; return value }(),
		func() domain.Values { value := base; value.ResidualValueCents = 101; return value }(),
		func() domain.Values { value := base; value.MaintenanceReservePerHourCents = -1; return value }(),
		func() domain.Values { value := base; value.Status = "offline"; return value }(),
	}
	for _, input := range tests {
		if _, err := service.Create(context.Background(), input); !errors.Is(err, ErrInvalidPrinter) {
			t.Fatalf("Create(%#v)=%v", input, err)
		}
	}
}

type repositoryStub struct {
	values domain.Values
	at     time.Time
	err    error
}

func (stub *repositoryStub) Create(_ context.Context, values domain.Values, at time.Time) (domain.Printer, error) {
	stub.values, stub.at = values, at
	return domain.Printer{Name: values.Name, Manufacturer: values.Manufacturer, Model: values.Model, NozzleDiameter: values.NozzleDiameter, UsefulLifeHours: values.UsefulLifeHours, Status: values.Status}, stub.err
}
func (stub *repositoryStub) FindByID(context.Context, string) (domain.Printer, error) {
	return domain.Printer{}, stub.err
}
func (stub *repositoryStub) List(context.Context) ([]domain.Printer, error) {
	return []domain.Printer{}, stub.err
}
func (stub *repositoryStub) Update(context.Context, string, domain.Values, time.Time) (domain.Printer, error) {
	return domain.Printer{}, stub.err
}
func (stub *repositoryStub) Delete(context.Context, string) error { return stub.err }
