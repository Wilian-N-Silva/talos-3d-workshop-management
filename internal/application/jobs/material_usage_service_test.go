package jobs

import (
	"context"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
	"testing"
	"time"
)

func TestMaterialUsageSummaryUsesExactTotals(t *testing.T) {
	a := "2.25"
	repo := &usageRepoStub{items: []domain.MaterialUsage{{PlannedGrams: "10.125", ActualGrams: &a}, {PlannedGrams: "3.875"}}}
	service, _ := NewMaterialUsageService(repo)
	summary, err := service.List(context.Background(), jid)
	if err != nil || summary.TotalPlannedGrams != "14" || summary.TotalActualGrams != "2.25" {
		t.Fatalf("List()=%#v,%v", summary, err)
	}
}
func TestMaterialUsageAllowsSameSpoolForDifferentRoles(t *testing.T) {
	repo := &usageRepoStub{}
	service, _ := NewMaterialUsageService(repo)
	base := domain.MaterialUsageValues{SpoolID: "66666666-6666-4666-8666-666666666666", Role: domain.MaterialRoleModel, PlannedGrams: "10.000", MeasurementSource: domain.SourceSlicer}
	if _, err := service.Create(context.Background(), jid, base); err != nil {
		t.Fatal(err)
	}
	base.Role = domain.MaterialRoleSupport
	if _, err := service.Create(context.Background(), jid, base); err != nil || len(repo.created) != 2 {
		t.Fatalf("second role=%v created=%#v", err, repo.created)
	}
}

type usageRepoStub struct {
	items   []domain.MaterialUsage
	created []domain.MaterialUsageValues
}

func (r *usageRepoStub) CreateMaterialUsage(_ context.Context, _ string, v domain.MaterialUsageValues, _ time.Time) (domain.MaterialUsage, error) {
	r.created = append(r.created, v)
	return domain.MaterialUsage{PlannedGrams: v.PlannedGrams}, nil
}
func (r *usageRepoStub) ListMaterialUsage(context.Context, string) ([]domain.MaterialUsage, error) {
	return r.items, nil
}
func (r *usageRepoStub) UpdateMaterialUsage(context.Context, string, string, domain.MaterialUsageValues, time.Time) (domain.MaterialUsage, error) {
	return domain.MaterialUsage{}, nil
}
func (r *usageRepoStub) DeleteMaterialUsage(context.Context, string, string) error { return nil }
