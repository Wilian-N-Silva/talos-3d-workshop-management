package config

import (
	"strings"
	"testing"
)

func TestLoadUsesDefaults(t *testing.T) {
	config, err := load(environment(map[string]string{
		databaseURLEnvironment: "postgres://talos:password@postgres:5432/talos",
	}))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if config.ServerPort != defaultServerPort {
		t.Fatalf("ServerPort = %d, want %d", config.ServerPort, defaultServerPort)
	}
	if config.ListenAddress() != ":8080" {
		t.Fatalf("ListenAddress() = %q, want :8080", config.ListenAddress())
	}
	if config.DataDirectory != "data" {
		t.Fatalf("DataDirectory = %q, want data", config.DataDirectory)
	}
	if config.TrustedLAN {
		t.Fatal("TrustedLAN = true, want false")
	}
	if config.UploadMaxBytes != defaultUploadMaxBytes {
		t.Fatalf("UploadMaxBytes = %d, want %d", config.UploadMaxBytes, defaultUploadMaxBytes)
	}
	if config.DefaultLocale != defaultLocale || config.DefaultCurrency != defaultCurrency || config.DefaultTimezone != defaultTimezone {
		t.Fatal("localization defaults do not match the documented values")
	}
}

func TestLoadUsesConfiguredValues(t *testing.T) {
	config, err := load(environment(map[string]string{
		"TALOS_SERVER_PORT":      "9090",
		databaseURLEnvironment:   "postgresql://talos:password@db:5432/workshop",
		"TALOS_DATA_DIR":         "./workshop-data",
		"TALOS_TRUSTED_LAN":      "true",
		"TALOS_UPLOAD_MAX_BYTES": "2048",
		"TALOS_DEFAULT_LOCALE":   "en-US",
		"TALOS_DEFAULT_CURRENCY": "USD",
		"TALOS_DEFAULT_TIMEZONE": "UTC",
	}))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if config.ServerPort != 9090 || config.ListenAddress() != ":9090" {
		t.Fatal("configured server port was not applied")
	}
	if config.DataDirectory != "workshop-data" || !config.TrustedLAN || config.UploadMaxBytes != 2048 {
		t.Fatal("configured operational values were not applied")
	}
	if config.DefaultLocale != "en-US" || config.DefaultCurrency != "USD" || config.DefaultTimezone != "UTC" {
		t.Fatal("configured localization values were not applied")
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name      string
		values    map[string]string
		wantError string
	}{
		{name: "missing database URL", values: map[string]string{}, wantError: "TALOS_DATABASE_URL is required"},
		{name: "invalid port", values: validEnvironment("70000"), wantError: "TALOS_SERVER_PORT"},
		{name: "invalid database scheme", values: map[string]string{databaseURLEnvironment: "mysql://db/talos"}, wantError: "valid PostgreSQL URL"},
		{name: "invalid trusted LAN", values: withValue("TALOS_TRUSTED_LAN", "sometimes"), wantError: "TALOS_TRUSTED_LAN must be a boolean"},
		{name: "invalid upload size", values: withValue("TALOS_UPLOAD_MAX_BYTES", "0"), wantError: "TALOS_UPLOAD_MAX_BYTES"},
		{name: "invalid locale", values: withValue("TALOS_DEFAULT_LOCALE", "pt_br"), wantError: "TALOS_DEFAULT_LOCALE"},
		{name: "invalid currency", values: withValue("TALOS_DEFAULT_CURRENCY", "brl"), wantError: "TALOS_DEFAULT_CURRENCY"},
		{name: "invalid timezone", values: withValue("TALOS_DEFAULT_TIMEZONE", "Mars/Olympus"), wantError: "TALOS_DEFAULT_TIMEZONE"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := load(environment(test.values))
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("load() error = %v, want error containing %q", err, test.wantError)
			}
		})
	}
}

func TestDatabaseURLIsNotExposedInValidationErrors(t *testing.T) {
	const secret = "do-not-log-this-password"
	_, err := load(environment(map[string]string{
		databaseURLEnvironment: "mysql://talos:" + secret + "@postgres:5432/talos",
	}))
	if err == nil {
		t.Fatal("load() error = nil, want validation error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("validation error exposed database secret: %v", err)
	}
}

func environment(values map[string]string) environmentLookup {
	return func(name string) (string, bool) {
		value, ok := values[name]
		return value, ok
	}
}

func validEnvironment(port string) map[string]string {
	return map[string]string{
		"TALOS_SERVER_PORT":    port,
		databaseURLEnvironment: "postgres://talos:password@postgres:5432/talos",
	}
}

func withValue(name, value string) map[string]string {
	values := validEnvironment("8080")
	values[name] = value
	return values
}
