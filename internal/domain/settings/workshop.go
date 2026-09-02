// Package settings contains persisted workshop configuration value definitions.
package settings

import (
	"errors"
	"time"
)

// ErrWorkshopSettingsNotFound indicates that the singleton has not been initialized.
var ErrWorkshopSettingsNotFound = errors.New("workshop settings not found")

// Theme is one of the fixed Release 1 presentation modes.
type Theme string

const (
	ThemeLight  Theme = "light"
	ThemeDark   Theme = "dark"
	ThemeSystem Theme = "system"
)

// WorkshopSettings is the single persisted workshop identity and presentation policy.
type WorkshopSettings struct {
	WorkshopName    string
	LogoFileID      *string
	DefaultLocale   string
	DefaultCurrency string
	DisplayTimezone string
	DefaultTheme    Theme
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Values contains validated mutable workshop settings.
type Values struct {
	WorkshopName    string
	DefaultLocale   string
	DefaultCurrency string
	DisplayTimezone string
	DefaultTheme    Theme
}
