package jobs

import (
	"context"
	"errors"
	"fmt"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
	"math/big"
	"regexp"
	"strings"
	"time"
)

var ErrInvalidMaterialUsage = errors.New("invalid job material usage")
var usageDecimalPattern = regexp.MustCompile(`^[0-9]{1,12}(\.[0-9]{1,6})?$`)

type MaterialUsageRepository interface {
	CreateMaterialUsage(context.Context, string, domain.MaterialUsageValues, time.Time) (domain.MaterialUsage, error)
	ListMaterialUsage(context.Context, string) ([]domain.MaterialUsage, error)
	UpdateMaterialUsage(context.Context, string, string, domain.MaterialUsageValues, time.Time) (domain.MaterialUsage, error)
	DeleteMaterialUsage(context.Context, string, string) error
}
type MaterialUsageService struct {
	repository MaterialUsageRepository
	now        func() time.Time
}

func NewMaterialUsageService(r MaterialUsageRepository) (*MaterialUsageService, error) {
	if r == nil {
		return nil, ErrInvalidConfiguration
	}
	return &MaterialUsageService{repository: r, now: time.Now}, nil
}
func (s *MaterialUsageService) Create(ctx context.Context, jobID string, input domain.MaterialUsageValues) (domain.MaterialUsage, error) {
	jobID, ok := normalizeID(jobID)
	if !ok {
		return domain.MaterialUsage{}, domain.ErrJobNotFound
	}
	v, err := normalizeMaterialUsage(input)
	if err != nil {
		return domain.MaterialUsage{}, err
	}
	result, err := s.repository.CreateMaterialUsage(ctx, jobID, v, s.now().UTC())
	if err != nil {
		return domain.MaterialUsage{}, fmt.Errorf("create job material usage: %w", err)
	}
	return result, nil
}
func (s *MaterialUsageService) List(ctx context.Context, jobID string) (domain.MaterialUsageSummary, error) {
	jobID, ok := normalizeID(jobID)
	if !ok {
		return domain.MaterialUsageSummary{}, domain.ErrJobNotFound
	}
	items, err := s.repository.ListMaterialUsage(ctx, jobID)
	if err != nil {
		return domain.MaterialUsageSummary{}, fmt.Errorf("list job material usage: %w", err)
	}
	summary := domain.MaterialUsageSummary{Items: items, TotalPlannedGrams: "0", TotalActualGrams: "0"}
	planned := new(big.Rat)
	actual := new(big.Rat)
	for _, item := range items {
		p, pok := new(big.Rat).SetString(item.PlannedGrams)
		if !pok {
			return domain.MaterialUsageSummary{}, errors.New("invalid persisted planned grams")
		}
		planned.Add(planned, p)
		if item.ActualGrams != nil {
			a, aok := new(big.Rat).SetString(*item.ActualGrams)
			if !aok {
				return domain.MaterialUsageSummary{}, errors.New("invalid persisted actual grams")
			}
			actual.Add(actual, a)
		}
	}
	summary.TotalPlannedGrams = formatUsageDecimal(planned)
	summary.TotalActualGrams = formatUsageDecimal(actual)
	return summary, nil
}
func (s *MaterialUsageService) Update(ctx context.Context, jobID, id string, input domain.MaterialUsageValues) (domain.MaterialUsage, error) {
	jobID, jok := normalizeID(jobID)
	id, iok := normalizeID(id)
	if !jok {
		return domain.MaterialUsage{}, domain.ErrJobNotFound
	}
	if !iok {
		return domain.MaterialUsage{}, domain.ErrMaterialUsageNotFound
	}
	v, err := normalizeMaterialUsage(input)
	if err != nil {
		return domain.MaterialUsage{}, err
	}
	result, err := s.repository.UpdateMaterialUsage(ctx, jobID, id, v, s.now().UTC())
	if err != nil {
		return domain.MaterialUsage{}, fmt.Errorf("update job material usage: %w", err)
	}
	return result, nil
}
func (s *MaterialUsageService) Delete(ctx context.Context, jobID, id string) error {
	jobID, jok := normalizeID(jobID)
	id, iok := normalizeID(id)
	if !jok {
		return domain.ErrJobNotFound
	}
	if !iok {
		return domain.ErrMaterialUsageNotFound
	}
	if err := s.repository.DeleteMaterialUsage(ctx, jobID, id); err != nil {
		return fmt.Errorf("delete job material usage: %w", err)
	}
	return nil
}
func normalizeMaterialUsage(v domain.MaterialUsageValues) (domain.MaterialUsageValues, error) {
	var ok bool
	if v.SpoolID, ok = normalizeID(v.SpoolID); !ok {
		return domain.MaterialUsageValues{}, ErrInvalidMaterialUsage
	}
	v.PlannedGrams = normalizeUsageDecimal(v.PlannedGrams)
	if !validUsageDecimal(v.PlannedGrams) {
		return domain.MaterialUsageValues{}, ErrInvalidMaterialUsage
	}
	for _, value := range []**string{&v.ActualGrams, &v.PlannedMeters, &v.ActualMeters} {
		if *value != nil {
			normalized := normalizeUsageDecimal(**value)
			*value = &normalized
			if !validUsageDecimal(normalized) {
				return domain.MaterialUsageValues{}, ErrInvalidMaterialUsage
			}
		}
	}
	if !validMaterialRole(v.Role) || !validMeasurementSource(v.MeasurementSource) || (v.HistoricalMaterialCostCents != nil && *v.HistoricalMaterialCostCents < 0) || (v.ReplacementMaterialCostCents != nil && *v.ReplacementMaterialCostCents < 0) {
		return domain.MaterialUsageValues{}, ErrInvalidMaterialUsage
	}
	return v, nil
}
func validUsageDecimal(v string) bool {
	n, ok := new(big.Rat).SetString(v)
	return usageDecimalPattern.MatchString(v) && ok && n.Sign() >= 0
}
func normalizeUsageDecimal(v string) string {
	v = strings.TrimSpace(v)
	if strings.Contains(v, ".") {
		v = strings.TrimRight(strings.TrimRight(v, "0"), ".")
	}
	if v == "" {
		return "0"
	}
	return v
}
func formatUsageDecimal(v *big.Rat) string { return normalizeUsageDecimal(v.FloatString(6)) }
func validMaterialRole(v domain.MaterialRole) bool {
	return v == domain.MaterialRoleModel || v == domain.MaterialRoleSupport || v == domain.MaterialRolePurge || v == domain.MaterialRoleOther
}
func validMeasurementSource(v domain.MeasurementSource) bool {
	return v == domain.SourceSlicer || v == domain.SourceSpoolWeightDelta || v == domain.SourceManual || v == domain.SourcePrinter || v == domain.SourceEstimated
}
