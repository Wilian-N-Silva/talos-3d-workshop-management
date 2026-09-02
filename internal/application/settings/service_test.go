package settings

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
)

func TestServiceInitializesValidatedDefaults(t *testing.T) {
	repository := &settingsRepositoryStub{}
	service := newTestService(t, repository, time.Now())

	_, err := service.Initialize(context.Background(), domainsettings.Values{
		WorkshopName:    "  Prototype Lab  ",
		DefaultLocale:   "pt-BR",
		DefaultCurrency: "BRL",
		DisplayTimezone: "America/Sao_Paulo",
		DefaultTheme:    domainsettings.ThemeSystem,
	})
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if repository.initializeValues.WorkshopName != "Prototype Lab" || repository.initializeCalls != 1 {
		t.Fatalf("Initialize() values = %#v, calls = %d", repository.initializeValues, repository.initializeCalls)
	}
}

func TestServiceUpdatesValidatedSettingsAtUTC(t *testing.T) {
	now := time.Date(2026, time.September, 2, 10, 30, 0, 0, time.FixedZone("local", -3*60*60))
	repository := &settingsRepositoryStub{}
	service := newTestService(t, repository, now)

	_, err := service.Update(context.Background(), domainsettings.Values{
		WorkshopName:    "Design Studio",
		DefaultLocale:   "en-US",
		DefaultCurrency: "USD",
		DisplayTimezone: "UTC",
		DefaultTheme:    domainsettings.ThemeDark,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if repository.updateValues.DefaultTheme != domainsettings.ThemeDark || !repository.updatedAt.Equal(now.UTC()) {
		t.Fatalf("Update() values = %#v at %s", repository.updateValues, repository.updatedAt)
	}
}

func TestServiceRejectsInvalidSettingsBeforePersistence(t *testing.T) {
	valid := domainsettings.Values{
		WorkshopName:    "Workshop",
		DefaultLocale:   "pt-BR",
		DefaultCurrency: "BRL",
		DisplayTimezone: "America/Sao_Paulo",
		DefaultTheme:    domainsettings.ThemeSystem,
	}
	tests := []struct {
		name   string
		change func(*domainsettings.Values)
	}{
		{name: "empty name", change: func(value *domainsettings.Values) { value.WorkshopName = "  " }},
		{name: "long name", change: func(value *domainsettings.Values) {
			value.WorkshopName = strings.Repeat("a", maximumWorkshopNameLength+1)
		}},
		{name: "locale", change: func(value *domainsettings.Values) { value.DefaultLocale = "pt_br" }},
		{name: "currency", change: func(value *domainsettings.Values) { value.DefaultCurrency = "brl" }},
		{name: "timezone", change: func(value *domainsettings.Values) { value.DisplayTimezone = "Mars/Olympus" }},
		{name: "theme", change: func(value *domainsettings.Values) { value.DefaultTheme = "custom" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.change(&input)
			repository := &settingsRepositoryStub{}
			service := newTestService(t, repository, time.Now())
			if _, err := service.Update(context.Background(), input); !errors.Is(err, ErrInvalidWorkshopSettings) {
				t.Fatalf("Update() error = %v", err)
			}
			if repository.updateCalls != 0 {
				t.Fatalf("repository update calls = %d", repository.updateCalls)
			}
		})
	}
}

func TestServiceWrapsRepositoryErrors(t *testing.T) {
	dependencyError := errors.New("database unavailable")
	repository := &settingsRepositoryStub{getError: dependencyError}
	service := newTestService(t, repository, time.Now())
	if _, err := service.Get(context.Background()); !errors.Is(err, dependencyError) {
		t.Fatalf("Get() error = %v", err)
	}
}

func TestNewServiceRejectsInvalidConfiguration(t *testing.T) {
	if _, err := NewService(nil); !errors.Is(err, ErrInvalidWorkshopSettingsConfiguration) {
		t.Fatalf("nil repository error = %v", err)
	}
	if _, err := newService(&settingsRepositoryStub{}, nil); !errors.Is(err, ErrInvalidWorkshopSettingsConfiguration) {
		t.Fatalf("nil clock error = %v", err)
	}
}

func newTestService(t *testing.T, repository Repository, now time.Time) *Service {
	t.Helper()
	service, err := newService(repository, func() time.Time { return now })
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}
	return service
}

type settingsRepositoryStub struct {
	result           domainsettings.WorkshopSettings
	initializeValues domainsettings.Values
	initializeError  error
	initializeCalls  int
	getError         error
	getCalls         int
	updateValues     domainsettings.Values
	updatedAt        time.Time
	updateError      error
	updateCalls      int
}

func (stub *settingsRepositoryStub) Initialize(
	_ context.Context,
	values domainsettings.Values,
) (domainsettings.WorkshopSettings, error) {
	stub.initializeCalls++
	stub.initializeValues = values
	return stub.result, stub.initializeError
}

func (stub *settingsRepositoryStub) Get(context.Context) (domainsettings.WorkshopSettings, error) {
	stub.getCalls++
	return stub.result, stub.getError
}

func (stub *settingsRepositoryStub) Update(
	_ context.Context,
	values domainsettings.Values,
	updatedAt time.Time,
) (domainsettings.WorkshopSettings, error) {
	stub.updateCalls++
	stub.updateValues = values
	stub.updatedAt = updatedAt
	return stub.result, stub.updateError
}
