package maintenance

import (
	"context"
	"errors"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/maintenance"
	"math"
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

func TestMaintenanceValidatesHoursBeforeNormalization(t *testing.T) {
	for _, hours := range []string{"", " ", ".", "0.", "1..0", "-1", "1e2", "1/2", "1.0000", "1000000000000000"} {
		t.Run(hours, func(t *testing.T) {
			repository := &maintenanceRepositoryStub{}
			service, _ := NewService(repository)
			_, err := service.Create(context.Background(), maintenancePrinterID, maintenanceUserID, domain.Values{
				Type: domain.TypeInspection, PerformedAt: time.Now(), Description: "Inspect", PrinterHours: &hours,
			})
			if !errors.Is(err, ErrInvalidEvent) || repository.creates != 0 {
				t.Fatalf("invalid hours %q: error=%v writes=%d", hours, err, repository.creates)
			}
		})
	}
}

func TestMaintenanceOptionalValuesAndStorageBounds(t *testing.T) {
	for _, test := range []struct {
		name     string
		hours    *string
		cost     *int64
		downtime int
		valid    bool
	}{
		{name: "omitted", valid: true},
		{name: "zero", hours: maintenancePointer("0"), cost: maintenancePointer(int64(0)), valid: true},
		{name: "maximum", hours: maintenancePointer("999999999999999.999"), cost: maintenancePointer(int64(math.MaxInt64)), downtime: math.MaxInt32, valid: true},
		{name: "negative cost", cost: maintenancePointer(int64(-1))},
		{name: "negative downtime", downtime: -1},
		{name: "downtime overflow", downtime: math.MaxInt32 + 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository := &maintenanceRepositoryStub{}
			service, _ := NewService(repository)
			_, err := service.Create(context.Background(), maintenancePrinterID, maintenanceUserID, domain.Values{
				Type: domain.TypeInspection, PerformedAt: time.Now(), Description: "Inspect", PrinterHours: test.hours, CostCents: test.cost, DowntimeMinutes: test.downtime,
			})
			if test.valid {
				if err != nil || repository.creates != 1 {
					t.Fatalf("error=%v writes=%d", err, repository.creates)
				}
			} else if !errors.Is(err, ErrInvalidEvent) || repository.creates != 0 {
				t.Fatalf("error=%v writes=%d", err, repository.creates)
			}
		})
	}
}

func TestMaintenanceNormalizesAuditAndAllTypes(t *testing.T) {
	local := time.Date(2026, 9, 4, 12, 0, 0, 0, time.FixedZone("local", -3*60*60))
	for _, kind := range []domain.Type{domain.TypeCleaning, domain.TypePreventive, domain.TypeCorrective, domain.TypeReplacement, domain.TypeUpgrade, domain.TypeInspection} {
		t.Run(string(kind), func(t *testing.T) {
			repository := &maintenanceRepositoryStub{}
			service, _ := NewService(repository)
			service.now = func() time.Time { return local }
			event, err := service.Create(context.Background(), maintenancePrinterID, maintenanceUserID, domain.Values{
				Type: kind, PerformedAt: local, Description: " Inspect ", Notes: " Done ", PrinterHours: maintenancePointer(" 10.250 "),
			})
			if err != nil || event.CreatedAt.Location() != time.UTC || repository.values.PerformedAt.Location() != time.UTC || !repository.values.PerformedAt.Equal(local) || repository.values.Description != "Inspect" || repository.values.Notes != "Done" || *event.PrinterHours != "10.25" {
				t.Fatalf("event=%#v values=%#v error=%v", event, repository.values, err)
			}
		})
	}
}

func maintenancePointer[T any](value T) *T { return &value }

type maintenanceRepositoryStub struct {
	creates int
	values  domain.Values
}

func (repository *maintenanceRepositoryStub) Create(_ context.Context, printerID, createdBy string, v domain.Values, createdAt time.Time) (domain.Event, error) {
	repository.creates++
	repository.values = v
	return domain.Event{PrinterID: printerID, CreatedBy: createdBy, Type: v.Type, PrinterHours: v.PrinterHours, CostCents: v.CostCents, CreatedAt: createdAt}, nil
}
func (*maintenanceRepositoryStub) List(context.Context, string) ([]domain.Event, error) {
	return []domain.Event{}, nil
}
