package serverconnection

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "HTTP LAN", raw: "  HTTP://workshop.local:8080/  ", want: "http://workshop.local:8080"},
		{name: "HTTPS path", raw: "https://example.test/talos/", want: "https://example.test/talos"},
		{name: "IPv6", raw: "http://[::1]:8080", want: "http://[::1]:8080"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeBaseURL(test.raw)
			if err != nil || got != test.want {
				t.Fatalf("NormalizeBaseURL() = %q, %v; want %q", got, err, test.want)
			}
		})
	}
}

func TestNormalizeBaseURLRejectsUnsafeValues(t *testing.T) {
	for _, value := range []string{
		"", "workshop.local:8080", "ftp://workshop.local", "http://", "http://user:password@workshop.local",
		"http://workshop.local?token=value", "http://workshop.local/#fragment", "http://workshop.local:0",
		"http://workshop.local:70000", "http://workshop.local:invalid",
	} {
		t.Run(value, func(t *testing.T) {
			if _, err := NormalizeBaseURL(value); !errors.Is(err, ErrInvalidBaseURL) {
				t.Fatalf("NormalizeBaseURL(%q) error = %v", value, err)
			}
		})
	}
}

func TestStoreStartsEmptyAndPersistsNormalizedConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "connection.json")
	store := NewStore(path)
	configuration, err := store.Load()
	if err != nil || configuration.ServerBaseURL != "" {
		t.Fatalf("Load() before save = %#v, %v", configuration, err)
	}

	saved, err := store.Save(" https://workshop.local:8443/ ")
	if err != nil || saved.ServerBaseURL != "https://workshop.local:8443" {
		t.Fatalf("Save() = %#v, %v", saved, err)
	}
	loaded, err := store.Load()
	if err != nil || loaded != saved {
		t.Fatalf("Load() = %#v, %v; want %#v", loaded, err, saved)
	}
}

func TestStoreRejectsInvalidInputWithoutOverwritingConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connection.json")
	store := NewStore(path)
	if _, err := store.Save("http://workshop.local"); err != nil {
		t.Fatalf("initial Save() error = %v", err)
	}
	if _, err := store.Save("postgres://database.local/talos"); !errors.Is(err, ErrInvalidBaseURL) {
		t.Fatalf("invalid Save() error = %v", err)
	}
	loaded, err := store.Load()
	if err != nil || loaded.ServerBaseURL != "http://workshop.local" {
		t.Fatalf("Load() after rejected save = %#v, %v", loaded, err)
	}
}

func TestStoreRejectsMalformedPersistedConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connection.json")
	if err := os.WriteFile(path, []byte(`{"server_base_url":"postgres://database.local"}`), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, err := NewStore(path).Load(); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestStoreRejectsTrailingPersistedContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connection.json")
	if err := os.WriteFile(path, []byte("{\"server_base_url\":\"http://workshop.local\"}\n{}"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, err := NewStore(path).Load(); !errors.Is(err, ErrInvalidConfiguration) {
		t.Fatalf("Load() error = %v", err)
	}
}
