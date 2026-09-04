package maintenance

import (
	"context"
	"errors"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/maintenance"
	"testing"
	"time"
)

const maintenancePrinterID = "11111111-1111-4111-8111-111111111111"
const maintenanceUserID = "22222222-2222-4222-8222-222222222222"

func TestMaintenanceAcceptsOptionalCostAndPrinterHours(t *testing.T) {
	hours := "1500.250"
	cost := int64(7500)
	repository := &maintenanceRepositoryStub{}
	service, _ := NewService(repository)
	event, err := service.Create(context.Background(), maintenancePrinterID, maintenanceUserID, domain.Values{Type: domain.TypePreventive, PerformedAt: time.Now(), PrinterHours: &hours, Description: "Lubricate axes", CostCents: &cost, DowntimeMinutes: 30})
	if err != nil || event.PrinterHours == nil || *event.PrinterHours != "1500.25" || event.CostCents == nil || *event.CostCents != 7500 {
		t.Fatalf("Create()=%#v,%v", event, err)
	}
}
func TestMaintenanceRejectsInvalidValues(t *testing.T) {
	service, _ := NewService(&maintenanceRepositoryStub{})
	_, err := service.Create(context.Background(), maintenancePrinterID, maintenanceUserID, domain.Values{Type: domain.TypePreventive, PerformedAt: time.Now()})
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("Create()=%v", err)
	}
}

type maintenanceRepositoryStub struct{}

func (*maintenanceRepositoryStub) Create(_ context.Context, printerID, createdBy string, v domain.Values, createdAt time.Time) (domain.Event, error) {
	return domain.Event{PrinterID: printerID, CreatedBy: createdBy, Type: v.Type, PrinterHours: v.PrinterHours, CostCents: v.CostCents, CreatedAt: createdAt}, nil
}
func (*maintenanceRepositoryStub) List(context.Context, string) ([]domain.Event, error) {
	return []domain.Event{}, nil
}
