// Package settings implements workshop-settings validation and use cases.
package settings

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"

	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
)

const (
	maximumWorkshopNameLength = 200
	maximumTimezoneLength     = 100
)

var (
	// ErrInvalidWorkshopSettings indicates invalid user-controlled settings values.
	ErrInvalidWorkshopSettings = errors.New("invalid workshop settings")
	// ErrInvalidWorkshopSettingsConfiguration indicates a missing service dependency.
	ErrInvalidWorkshopSettingsConfiguration = errors.New("invalid workshop settings configuration")
	localePattern                           = regexp.MustCompile(`^[a-z]{2}-[A-Z]{2}$`)
	currencyPattern                         = regexp.MustCompile(`^[A-Z]{3}$`)
)

// Repository persists the singleton workshop settings record.
type Repository interface {
	Initialize(context.Context, domainsettings.Values) (domainsettings.WorkshopSettings, error)
	Get(context.Context) (domainsettings.WorkshopSettings, error)
	Update(context.Context, domainsettings.Values, time.Time) (domainsettings.WorkshopSettings, error)
}

// Service validates and manages workshop settings.
type Service struct {
	repository Repository
	now        func() time.Time
}

// NewService creates a workshop-settings service.
func NewService(repository Repository) (*Service, error) {
	return newService(repository, time.Now)
}

func newService(repository Repository, now func() time.Time) (*Service, error) {
	if repository == nil || now == nil {
		return nil, ErrInvalidWorkshopSettingsConfiguration
	}
	return &Service{repository: repository, now: now}, nil
}

// Initialize creates the singleton with process-configured defaults only when
// it does not already exist.
func (service *Service) Initialize(
	ctx context.Context,
	defaults domainsettings.Values,
) (domainsettings.WorkshopSettings, error) {
	values, err := normalizeValues(defaults)
	if err != nil {
		return domainsettings.WorkshopSettings{}, err
	}
	settings, err := service.repository.Initialize(ctx, values)
	if err != nil {
		return domainsettings.WorkshopSettings{}, fmt.Errorf("initialize workshop settings: %w", err)
	}
	return settings, nil
}

// Get returns the persisted singleton.
func (service *Service) Get(ctx context.Context) (domainsettings.WorkshopSettings, error) {
	settings, err := service.repository.Get(ctx)
	if err != nil {
		return domainsettings.WorkshopSettings{}, fmt.Errorf("get workshop settings: %w", err)
	}
	return settings, nil
}

// Update validates and replaces the mutable settings fields. Logo association
// remains exclusive to the dedicated logo workflow.
func (service *Service) Update(
	ctx context.Context,
	input domainsettings.Values,
) (domainsettings.WorkshopSettings, error) {
	values, err := normalizeValues(input)
	if err != nil {
		return domainsettings.WorkshopSettings{}, err
	}
	settings, err := service.repository.Update(ctx, values, service.now().UTC())
	if err != nil {
		return domainsettings.WorkshopSettings{}, fmt.Errorf("update workshop settings: %w", err)
	}
	return settings, nil
}

func normalizeValues(input domainsettings.Values) (domainsettings.Values, error) {
	values := domainsettings.Values{
		WorkshopName:    strings.TrimSpace(input.WorkshopName),
		DefaultLocale:   strings.TrimSpace(input.DefaultLocale),
		DefaultCurrency: strings.TrimSpace(input.DefaultCurrency),
		DisplayTimezone: strings.TrimSpace(input.DisplayTimezone),
		DefaultTheme:    input.DefaultTheme,
	}
	if len(values.WorkshopName) == 0 || len(values.WorkshopName) > maximumWorkshopNameLength ||
		!localePattern.MatchString(values.DefaultLocale) ||
		!currencyPattern.MatchString(values.DefaultCurrency) ||
		len(values.DisplayTimezone) == 0 || len(values.DisplayTimezone) > maximumTimezoneLength {
		return domainsettings.Values{}, ErrInvalidWorkshopSettings
	}
	if _, err := time.LoadLocation(values.DisplayTimezone); err != nil {
		return domainsettings.Values{}, ErrInvalidWorkshopSettings
	}
	switch values.DefaultTheme {
	case domainsettings.ThemeLight, domainsettings.ThemeDark, domainsettings.ThemeSystem:
	default:
		return domainsettings.Values{}, ErrInvalidWorkshopSettings
	}
	return values, nil
}
