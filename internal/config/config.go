package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

const (
	defaultServerPort      = 8080
	defaultDataDirectory   = "./data"
	defaultUploadMaxBytes  = 100 * 1024 * 1024
	defaultLocale          = "pt-BR"
	defaultCurrency        = "BRL"
	defaultTimezone        = "America/Sao_Paulo"
	databaseURLEnvironment = "TALOS_DATABASE_URL"
)

var (
	localePattern   = regexp.MustCompile(`^[a-z]{2}-[A-Z]{2}$`)
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
)

// Config contains the server's process-level environment configuration.
type Config struct {
	ServerPort      int
	DatabaseURL     string
	DataDirectory   string
	TrustedLAN      bool
	UploadMaxBytes  int64
	DefaultLocale   string
	DefaultCurrency string
	DefaultTimezone string
}

// Load reads and validates server configuration from the process environment.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

// ListenAddress returns the TCP address used by the HTTP server.
func (c Config) ListenAddress() string {
	return fmt.Sprintf(":%d", c.ServerPort)
}

type environmentLookup func(string) (string, bool)

func load(lookup environmentLookup) (Config, error) {
	serverPort, err := integer(lookup, "TALOS_SERVER_PORT", defaultServerPort, 1, 65535)
	if err != nil {
		return Config{}, err
	}

	databaseURL, err := requiredPostgresURL(lookup)
	if err != nil {
		return Config{}, err
	}

	dataDirectory := valueOrDefault(lookup, "TALOS_DATA_DIR", defaultDataDirectory)
	if dataDirectory == "" {
		return Config{}, fmt.Errorf("TALOS_DATA_DIR must not be empty")
	}

	trustedLAN, err := boolean(lookup, "TALOS_TRUSTED_LAN", false)
	if err != nil {
		return Config{}, err
	}

	uploadMaxBytes, err := integer64(lookup, "TALOS_UPLOAD_MAX_BYTES", defaultUploadMaxBytes, 1)
	if err != nil {
		return Config{}, err
	}

	locale := valueOrDefault(lookup, "TALOS_DEFAULT_LOCALE", defaultLocale)
	if !localePattern.MatchString(locale) {
		return Config{}, fmt.Errorf("TALOS_DEFAULT_LOCALE must use language-REGION format, for example pt-BR")
	}

	currency := valueOrDefault(lookup, "TALOS_DEFAULT_CURRENCY", defaultCurrency)
	if !currencyPattern.MatchString(currency) {
		return Config{}, fmt.Errorf("TALOS_DEFAULT_CURRENCY must be a three-letter uppercase currency code")
	}

	timezone := valueOrDefault(lookup, "TALOS_DEFAULT_TIMEZONE", defaultTimezone)
	if _, err := time.LoadLocation(timezone); err != nil {
		return Config{}, fmt.Errorf("TALOS_DEFAULT_TIMEZONE must be a valid IANA timezone")
	}

	return Config{
		ServerPort:      serverPort,
		DatabaseURL:     databaseURL,
		DataDirectory:   filepath.Clean(dataDirectory),
		TrustedLAN:      trustedLAN,
		UploadMaxBytes:  uploadMaxBytes,
		DefaultLocale:   locale,
		DefaultCurrency: currency,
		DefaultTimezone: timezone,
	}, nil
}

func requiredPostgresURL(lookup environmentLookup) (string, error) {
	raw, ok := lookup(databaseURLEnvironment)
	value := strings.TrimSpace(raw)
	if !ok || value == "" {
		return "", fmt.Errorf("%s is required", databaseURLEnvironment)
	}

	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
		return "", fmt.Errorf("%s must be a valid PostgreSQL URL", databaseURLEnvironment)
	}

	return value, nil
}

func integer(lookup environmentLookup, name string, fallback, minimum, maximum int) (int, error) {
	raw := valueOrDefault(lookup, name, strconv.Itoa(fallback))
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("%s must be an integer between %d and %d", name, minimum, maximum)
	}

	return value, nil
}

func integer64(lookup environmentLookup, name string, fallback, minimum int64) (int64, error) {
	raw := valueOrDefault(lookup, name, strconv.FormatInt(fallback, 10))
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < minimum {
		return 0, fmt.Errorf("%s must be an integer greater than or equal to %d", name, minimum)
	}

	return value, nil
}

func boolean(lookup environmentLookup, name string, fallback bool) (bool, error) {
	raw := valueOrDefault(lookup, name, strconv.FormatBool(fallback))
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean", name)
	}

	return value, nil
}

func valueOrDefault(lookup environmentLookup, name, fallback string) string {
	value, ok := lookup(name)
	if !ok {
		return fallback
	}

	return strings.TrimSpace(value)
}
