package energy

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/energy"
)

const (
	energyJobID  = "11111111-1111-4111-8111-111111111111"
	energyUserID = "22222222-2222-4222-8222-222222222222"
)

func TestManualMeterDerivesExactMeasuredKWh(t *testing.T) {
	start, end := "120.125000", "121.375"
	repository := &energyRepositoryStub{}
	service, _ := NewService(repository)
	measurement, err := service.Create(context.Background(), energyJobID, energyUserID, domain.Values{Source: domain.SourceManualMeter, MeterStartKWh: &start, MeterEndKWh: &end, EnergyRateCentsPerKWh: 95, OccurredAt: time.Date(2026, 9, 4, 12, 0, 0, 0, time.FixedZone("local", -3*60*60))})
	if err != nil || measurement.MeasuredKWh == nil || *measurement.MeasuredKWh != "1.25" || repository.values.OccurredAt.Location() != time.UTC {
		t.Fatalf("Create()=%#v,%v values=%#v", measurement, err, repository.values)
	}
}

func TestEstimatedMeasurementKeepsPowerWithoutInventingKWh(t *testing.T) {
	power := "125.500"
	repository := &energyRepositoryStub{}
	service, _ := NewService(repository)
	measurement, err := service.Create(context.Background(), energyJobID, energyUserID, domain.Values{Source: domain.SourceEstimated, EstimatedAveragePowerW: &power, EnergyRateCentsPerKWh: 100, OccurredAt: time.Now()})
	if err != nil || measurement.EstimatedAveragePowerW == nil || *measurement.EstimatedAveragePowerW != "125.5" || measurement.MeasuredKWh != nil {
		t.Fatalf("Create()=%#v,%v", measurement, err)
	}
}

func TestEnergySourceEvidenceIsValidated(t *testing.T) {
	service, _ := NewService(&energyRepositoryStub{})
	tests := []domain.Values{
		{Source: domain.SourceManualMeter, OccurredAt: time.Now()},
		{Source: domain.SourceSmartPlug, OccurredAt: time.Now()},
		{Source: domain.SourceEstimated, OccurredAt: time.Now()},
	}
	for _, input := range tests {
		if _, err := service.Create(context.Background(), energyJobID, energyUserID, input); !errors.Is(err, ErrInvalidMeasurement) {
			t.Fatalf("Create(%s) error=%v, want invalid", input.Source, err)
		}
	}
}

type energyRepositoryStub struct{ values domain.Values }

func (repository *energyRepositoryStub) Create(_ context.Context, jobID, recordedBy string, values domain.Values) (domain.Measurement, error) {
	repository.values = values
	return domain.Measurement{JobID: jobID, RecordedBy: recordedBy, Source: values.Source, MeterStartKWh: values.MeterStartKWh, MeterEndKWh: values.MeterEndKWh, MeasuredKWh: values.MeasuredKWh, EstimatedAveragePowerW: values.EstimatedAveragePowerW, EnergyRateCentsPerKWh: values.EnergyRateCentsPerKWh, OccurredAt: values.OccurredAt}, nil
}
func (repository *energyRepositoryStub) List(context.Context, string) ([]domain.Measurement, error) {
	return []domain.Measurement{}, nil
}
