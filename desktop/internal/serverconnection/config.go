// Package serverconnection validates and persists the desktop's server endpoint.
package serverconnection

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

const (
	configurationDirectory = "TalosWorkshopManagement"
	configurationFile      = "connection.json"
)

var (
	// ErrInvalidBaseURL indicates a server URL that cannot safely identify an HTTP API origin.
	ErrInvalidBaseURL = errors.New("invalid server base URL")
	// ErrInvalidConfiguration indicates unreadable or malformed persisted connection state.
	ErrInvalidConfiguration = errors.New("invalid server connection configuration")
)

// Configuration is the non-secret local server connection state exposed to Wails.
type Configuration struct {
	ServerBaseURL string `json:"server_base_url"`
}

// Store persists connection state in one user-scoped JSON file.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewDefaultStore locates the current user's platform configuration directory.
func NewDefaultStore() (*Store, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("locate user configuration directory: %w", err)
	}
	return NewStore(filepath.Join(root, configurationDirectory, configurationFile)), nil
}

// NewStore creates a connection store at an explicit path, primarily for tests.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Load returns an empty configuration when the desktop has not been configured yet.
func (store *Store) Load() (Configuration, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	content, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return Configuration{}, nil
	}
	if err != nil {
		return Configuration{}, fmt.Errorf("read server connection configuration: %w", err)
	}
	var configuration Configuration
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&configuration); err != nil {
		return Configuration{}, fmt.Errorf("%w: %v", ErrInvalidConfiguration, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Configuration{}, fmt.Errorf("%w: trailing content", ErrInvalidConfiguration)
	}
	normalized, err := NormalizeBaseURL(configuration.ServerBaseURL)
	if err != nil {
		return Configuration{}, fmt.Errorf("%w: %v", ErrInvalidConfiguration, err)
	}
	configuration.ServerBaseURL = normalized
	return configuration, nil
}

// Save validates and persists the non-secret server endpoint.
func (store *Store) Save(rawBaseURL string) (Configuration, error) {
	normalized, err := NormalizeBaseURL(rawBaseURL)
	if err != nil {
		return Configuration{}, err
	}
	configuration := Configuration{ServerBaseURL: normalized}
	content, err := json.MarshalIndent(configuration, "", "  ")
	if err != nil {
		return Configuration{}, fmt.Errorf("encode server connection configuration: %w", err)
	}
	content = append(content, '\n')

	store.mu.Lock()
	defer store.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(store.path), 0o700); err != nil {
		return Configuration{}, fmt.Errorf("create server connection configuration directory: %w", err)
	}
	file, err := os.OpenFile(store.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return Configuration{}, fmt.Errorf("open server connection configuration: %w", err)
	}
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		return Configuration{}, fmt.Errorf("write server connection configuration: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return Configuration{}, fmt.Errorf("flush server connection configuration: %w", err)
	}
	if err := file.Close(); err != nil {
		return Configuration{}, fmt.Errorf("close server connection configuration: %w", err)
	}
	return configuration, nil
}

// NormalizeBaseURL accepts only credential-free absolute HTTP(S) server URLs.
func NormalizeBaseURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	parsed, err := url.Parse(value)
	if err != nil || parsed.Opaque != "" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", ErrInvalidBaseURL
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", ErrInvalidBaseURL
	}
	if parsed.Hostname() == "" {
		return "", ErrInvalidBaseURL
	}
	if port := parsed.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", ErrInvalidBaseURL
		}
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawPath = strings.TrimRight(parsed.RawPath, "/")
	return strings.TrimRight(parsed.String(), "/"), nil
}
