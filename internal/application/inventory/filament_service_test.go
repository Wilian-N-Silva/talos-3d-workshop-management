package inventory

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/inventory"
)

const inventoryID = "11111111-1111-4111-8111-111111111111"
const inventoryActorID = "22222222-2222-4222-8222-222222222222"

func TestFilamentServicePreservesExactDecimalsAndNormalizesInput(t *testing.T) {
	repository := &filamentRepositoryStub{}
	now := time.Date(2026, 9, 4, 12, 0, 0, 0, time.FixedZone("local", -3*60*60))
	service := &FilamentService{repository: repository, now: func() time.Time { return now }}
	color := " #aabbcc "
	material, err := service.CreateMaterial(context.Background(), domain.MaterialValues{Manufacturer: " Voolt3D ", Name: " Velvet ", MaterialType: " PLA ", ColorHex: &color, NominalDensity: "1.240000", DefaultReplacementCostPerKgCents: 12990})
	if err != nil || repository.material.Manufacturer != "Voolt3D" || repository.material.NominalDensity != "1.24" || repository.material.ColorHex == nil || *repository.material.ColorHex != "#AABBCC" || material.NominalDensity != "1.24" {
		t.Fatalf("CreateMaterial()=%#v,%v; values=%#v", material, err, repository.material)
	}
	spool, err := service.CreateSpool(context.Background(), domain.SpoolValues{Code: " A-001 ", MaterialID: inventoryID, NominalNetWeightG: "1000", TareWeightG: "250.000", Status: domain.SpoolSealed})
	if err != nil || repository.spool.NominalNetWeightG != "1000" || repository.spool.TareWeightG != "250" || spool.Code != "A-001" || !repository.at.Equal(now.UTC()) {
		t.Fatalf("CreateSpool()=%#v,%v; values=%#v at=%v", spool, err, repository.spool, repository.at)
	}
}

func TestFilamentServiceValidatesMaterialSpoolAndMeasurement(t *testing.T) {
	service, _ := NewFilamentService(&filamentRepositoryStub{})
	invalidMaterials := []domain.MaterialValues{{Manufacturer: "", Name: "PLA", MaterialType: "PLA", NominalDensity: "1.24"}, {Manufacturer: "Maker", Name: "PLA", MaterialType: "PLA", NominalDensity: "0"}, {Manufacturer: "Maker", Name: "PLA", MaterialType: "PLA", NominalDensity: "1,24"}}
	for _, value := range invalidMaterials {
		if _, err := service.CreateMaterial(context.Background(), value); !errors.Is(err, ErrInvalidMaterial) {
			t.Fatalf("material %#v error=%v", value, err)
		}
	}
	if _, err := service.CreateSpool(context.Background(), domain.SpoolValues{Code: "A", MaterialID: inventoryID, NominalNetWeightG: "1000", TareWeightG: "250", GrossWeightAtOpenG: stringPtr("200"), Status: domain.SpoolOpen}); !errors.Is(err, ErrInvalidSpool) {
		t.Fatalf("spool error=%v", err)
	}
	if _, err := service.RecordMeasurement(context.Background(), inventoryID, inventoryActorID, domain.MeasurementValues{MeasuredAt: time.Now(), GrossWeightG: "1.2345", Source: domain.MeasurementManual}); !errors.Is(err, ErrInvalidMeasurement) {
		t.Fatalf("measurement error=%v", err)
	}
}

func TestRecordMeasurementUsesActorAndUTC(t *testing.T) {
	repository := &filamentRepositoryStub{}
	service, _ := NewFilamentService(repository)
	measured := time.Date(2026, 9, 4, 9, 0, 0, 0, time.FixedZone("local", -3*60*60))
	_, err := service.RecordMeasurement(context.Background(), inventoryID, inventoryActorID, domain.MeasurementValues{MeasuredAt: measured, GrossWeightG: "845.500", Source: domain.MeasurementManual})
	if err != nil || repository.actor != inventoryActorID || repository.measurement.GrossWeightG != "845.5" || !repository.measurement.MeasuredAt.Equal(measured.UTC()) {
		t.Fatalf("RecordMeasurement() err=%v actor=%q values=%#v", err, repository.actor, repository.measurement)
	}
}

func stringPtr(value string) *string { return &value }

type filamentRepositoryStub struct {
	material    domain.MaterialValues
	spool       domain.SpoolValues
	measurement domain.MeasurementValues
	actor       string
	at          time.Time
	err         error
}

func (s *filamentRepositoryStub) CreateMaterial(_ context.Context, v domain.MaterialValues, at time.Time) (domain.Material, error) {
	s.material, s.at = v, at
	return domain.Material{Manufacturer: v.Manufacturer, NominalDensity: v.NominalDensity}, s.err
}
func (s *filamentRepositoryStub) FindMaterial(context.Context, string) (domain.Material, error) {
	return domain.Material{}, s.err
}
func (s *filamentRepositoryStub) ListMaterials(context.Context) ([]domain.Material, error) {
	return []domain.Material{}, s.err
}
func (s *filamentRepositoryStub) UpdateMaterial(context.Context, string, domain.MaterialValues, time.Time) (domain.Material, error) {
	return domain.Material{}, s.err
}
func (s *filamentRepositoryStub) DeleteMaterial(context.Context, string) error { return s.err }
func (s *filamentRepositoryStub) CreateSpool(_ context.Context, v domain.SpoolValues, at time.Time) (domain.Spool, error) {
	s.spool, s.at = v, at
	return domain.Spool{Code: v.Code}, s.err
}
func (s *filamentRepositoryStub) FindSpool(context.Context, string) (domain.Spool, error) {
	return domain.Spool{}, s.err
}
func (s *filamentRepositoryStub) ListSpools(context.Context) ([]domain.Spool, error) {
	return []domain.Spool{}, s.err
}
func (s *filamentRepositoryStub) UpdateSpool(context.Context, string, domain.SpoolValues, time.Time) (domain.Spool, error) {
	return domain.Spool{}, s.err
}
func (s *filamentRepositoryStub) DeleteSpool(context.Context, string) error { return s.err }
func (s *filamentRepositoryStub) RecordMeasurement(_ context.Context, _ string, actor string, v domain.MeasurementValues, at time.Time) (domain.SpoolMeasurement, error) {
	s.actor, s.measurement, s.at = actor, v, at
	return domain.SpoolMeasurement{}, s.err
}
func (s *filamentRepositoryStub) ListMeasurements(context.Context, string) ([]domain.SpoolMeasurement, error) {
	return []domain.SpoolMeasurement{}, s.err
}
