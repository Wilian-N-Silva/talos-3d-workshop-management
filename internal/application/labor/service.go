package labor

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	domainjobs "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/jobs"
	domain "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/labor"
)

var (
	ErrInvalidConfiguration = errors.New("invalid labor service configuration")
	ErrInvalidRate          = errors.New("invalid labor rate")
	ErrInvalidEntry         = errors.New("invalid labor entry")
)
var laborIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type Repository interface {
	CreateRate(context.Context, domain.RateValues, time.Time) (domain.Rate, error)
	ListRates(context.Context) ([]domain.Rate, error)
	UpdateRate(context.Context, string, domain.RateValues, time.Time) (domain.Rate, error)
	CreateEntry(context.Context, string, string, domain.EntryValues, time.Time) (domain.Entry, error)
	ListEntries(context.Context, string) ([]domain.Entry, error)
}
type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) (*Service, error) {
	if repository == nil {
		return nil, ErrInvalidConfiguration
	}
	return &Service{repository: repository, now: time.Now}, nil
}
func (service *Service) CreateRate(ctx context.Context, input domain.RateValues) (domain.Rate, error) {
	values, err := normalizeRate(input)
	if err != nil {
		return domain.Rate{}, err
	}
	rate, err := service.repository.CreateRate(ctx, values, service.now().UTC())
	if err != nil {
		return domain.Rate{}, fmt.Errorf("create labor rate: %w", err)
	}
	return rate, nil
}
func (service *Service) ListRates(ctx context.Context) ([]domain.Rate, error) {
	rates, err := service.repository.ListRates(ctx)
	if err != nil {
		return nil, fmt.Errorf("list labor rates: %w", err)
	}
	return rates, nil
}
func (service *Service) UpdateRate(ctx context.Context, id string, input domain.RateValues) (domain.Rate, error) {
	id, ok := normalizeID(id)
	if !ok {
		return domain.Rate{}, domain.ErrRateNotFound
	}
	values, err := normalizeRate(input)
	if err != nil {
		return domain.Rate{}, err
	}
	rate, err := service.repository.UpdateRate(ctx, id, values, service.now().UTC())
	if err != nil {
		return domain.Rate{}, fmt.Errorf("update labor rate: %w", err)
	}
	return rate, nil
}
func (service *Service) CreateEntry(ctx context.Context, jobID, recordedBy string, input domain.EntryValues) (domain.Entry, error) {
	jobID, jok := normalizeID(jobID)
	recordedBy, uok := normalizeID(recordedBy)
	input.LaborRateID, _ = normalizeID(input.LaborRateID)
	input.Notes = strings.TrimSpace(input.Notes)
	if !jok {
		return domain.Entry{}, domainjobs.ErrJobNotFound
	}
	if !uok || !laborIDPattern.MatchString(input.LaborRateID) || input.Minutes <= 0 || input.OccurredAt.IsZero() || len(input.Notes) > 10000 {
		return domain.Entry{}, ErrInvalidEntry
	}
	input.OccurredAt = input.OccurredAt.UTC()
	entry, err := service.repository.CreateEntry(ctx, jobID, recordedBy, input, service.now().UTC())
	if err != nil {
		return domain.Entry{}, fmt.Errorf("create labor entry: %w", err)
	}
	return entry, nil
}
func (service *Service) ListEntries(ctx context.Context, jobID string) (domain.Summary, error) {
	jobID, ok := normalizeID(jobID)
	if !ok {
		return domain.Summary{}, domainjobs.ErrJobNotFound
	}
	items, err := service.repository.ListEntries(ctx, jobID)
	if err != nil {
		return domain.Summary{}, fmt.Errorf("list labor entries: %w", err)
	}
	summary := domain.Summary{Items: items, MinutesByActivity: map[domain.ActivityType]int64{}}
	for _, item := range items {
		summary.TotalMinutes += int64(item.Minutes)
		summary.MinutesByActivity[item.ActivityType] += int64(item.Minutes)
	}
	return summary, nil
}
func normalizeRate(values domain.RateValues) (domain.RateValues, error) {
	values.Name = strings.TrimSpace(values.Name)
	if len(values.Name) == 0 || len(values.Name) > 200 || values.CostHourlyRateCents < 0 || !validActivity(values.ActivityType) {
		return domain.RateValues{}, ErrInvalidRate
	}
	return values, nil
}
func normalizeID(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	return value, laborIDPattern.MatchString(value)
}
func validActivity(value domain.ActivityType) bool {
	switch value {
	case domain.ActivitySetup, domain.ActivityMaterialHandling, domain.ActivitySupportRemoval, domain.ActivityFinishing, domain.ActivityPainting, domain.ActivityAssembly, domain.ActivityPackaging, domain.ActivityModeling, domain.ActivityCustomization, domain.ActivityConsulting, domain.ActivityOther:
		return true
	}
	return false
}
