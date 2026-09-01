package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/config"
)

func TestOpenConnectsToPostgreSQL(t *testing.T) {
	databaseURL := os.Getenv("TALOS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TALOS_TEST_DATABASE_URL is not set")
	}

	database, err := Open(context.Background(), testConfig(databaseURL))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	var value int
	if err := database.QueryRowContext(context.Background(), "SELECT 1").Scan(&value); err != nil {
		t.Fatalf("query PostgreSQL: %v", err)
	}
	if value != 1 {
		t.Fatalf("SELECT 1 = %d, want 1", value)
	}
	if database.Stats().MaxOpenConnections != 4 {
		t.Fatalf("MaxOpenConnections = %d, want 4", database.Stats().MaxOpenConnections)
	}
}

func TestOpenReturnsSecretSafeErrorWhenUnavailable(t *testing.T) {
	const secret = "do-not-log-this-password"
	serverConfig := testConfig("postgres://talos:" + secret + "@127.0.0.1:1/talos?sslmode=disable")
	serverConfig.DatabasePingTimeout = 250 * time.Millisecond

	database, err := Open(context.Background(), serverConfig)
	if database != nil {
		_ = database.Close()
		t.Fatal("Open() database is not nil")
	}
	if err == nil {
		t.Fatal("Open() error = nil")
	}
	if !strings.Contains(err.Error(), "connect to PostgreSQL") {
		t.Fatalf("Open() error = %q, want clear PostgreSQL failure", err)
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("Open() error exposed database secret: %v", err)
	}
}

func testConfig(databaseURL string) config.Config {
	return config.Config{
		DatabaseURL:                   databaseURL,
		DatabaseMaxOpenConnections:    4,
		DatabaseMaxIdleConnections:    2,
		DatabaseConnectionMaxLifetime: time.Minute,
		DatabaseConnectionMaxIdleTime: time.Minute,
		DatabasePingTimeout:           time.Second,
	}
}
