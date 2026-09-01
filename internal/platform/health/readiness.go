// Package health coordinates server health probes independently of HTTP.
package health

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

// Probe checks one required server dependency.
type Probe func(context.Context) error

// Readiness checks only the dependencies required to serve workshop requests.
// Printer state is intentionally absent because it must never affect server
// readiness.
type Readiness struct {
	database   Probe
	migrations Probe
	storage    Probe
	timeout    time.Duration
}

// NewReadiness constructs the required readiness dependency set.
func NewReadiness(database, migrations, storage Probe, timeout time.Duration) *Readiness {
	return &Readiness{
		database:   database,
		migrations: migrations,
		storage:    storage,
		timeout:    timeout,
	}
}

// Check returns nil only when every required dependency is ready.
func (readiness *Readiness) Check(ctx context.Context) error {
	checkContext, cancel := context.WithTimeout(ctx, readiness.timeout)
	defer cancel()

	checks := []struct {
		name  string
		probe Probe
	}{
		{name: "PostgreSQL", probe: readiness.database},
		{name: "migrations", probe: readiness.migrations},
		{name: "storage", probe: readiness.storage},
	}

	for _, check := range checks {
		if err := check.probe(checkContext); err != nil {
			return fmt.Errorf("%s readiness check: %w", check.name, err)
		}
	}

	return nil
}

// StorageDirectory returns a probe that verifies the directory exists and
// supports the create, close, and remove operations required by file storage.
func StorageDirectory(directory string) Probe {
	return func(ctx context.Context) (returnErr error) {
		if err := ctx.Err(); err != nil {
			return err
		}

		info, err := os.Stat(directory)
		if err != nil {
			return fmt.Errorf("inspect data directory: %w", err)
		}
		if !info.IsDir() {
			return fmt.Errorf("data directory path is not a directory")
		}

		probeFile, err := os.CreateTemp(directory, ".talos-readiness-*")
		if err != nil {
			return fmt.Errorf("create storage probe: %w", err)
		}
		probePath := probeFile.Name()
		defer func() {
			if err := os.Remove(probePath); err != nil && !errors.Is(err, os.ErrNotExist) {
				returnErr = errors.Join(returnErr, fmt.Errorf("remove storage probe: %w", err))
			}
		}()

		if err := probeFile.Close(); err != nil {
			return fmt.Errorf("close storage probe: %w", err)
		}

		return ctx.Err()
	}
}
