package health

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadinessChecksOnlyRequiredDependencies(t *testing.T) {
	calls := map[string]int{}
	probe := func(name string) Probe {
		return func(context.Context) error {
			calls[name]++
			return nil
		}
	}

	readiness := NewReadiness(
		probe("database"),
		probe("migrations"),
		probe("storage"),
		time.Second,
	)

	if err := readiness.Check(context.Background()); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	for _, name := range []string{"database", "migrations", "storage"} {
		if calls[name] != 1 {
			t.Fatalf("%s probe calls = %d, want 1", name, calls[name])
		}
	}
}

func TestReadinessFailsWhenRequiredDependencyFails(t *testing.T) {
	probeError := errors.New("unavailable")
	ready := func(context.Context) error { return nil }
	failing := func(context.Context) error { return probeError }

	tests := []struct {
		name       string
		database   Probe
		migrations Probe
		storage    Probe
	}{
		{name: "database", database: failing, migrations: ready, storage: ready},
		{name: "migrations", database: ready, migrations: failing, storage: ready},
		{name: "storage", database: ready, migrations: ready, storage: failing},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			readiness := NewReadiness(test.database, test.migrations, test.storage, time.Second)
			if err := readiness.Check(context.Background()); !errors.Is(err, probeError) {
				t.Fatalf("Check() error = %v, want wrapped probe error", err)
			}
		})
	}
}

func TestReadinessBoundsProbeDuration(t *testing.T) {
	waitForCancellation := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}
	ready := func(context.Context) error { return nil }

	readiness := NewReadiness(waitForCancellation, ready, ready, 10*time.Millisecond)
	if err := readiness.Check(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Check() error = %v, want deadline exceeded", err)
	}
}

func TestStorageDirectoryProbe(t *testing.T) {
	directory := t.TempDir()
	probe := StorageDirectory(directory)

	if err := probe(context.Background()); err != nil {
		t.Fatalf("storage probe error = %v", err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read temp directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("storage probe left %d files behind, want none", len(entries))
	}
}

func TestStorageDirectoryProbeRejectsInvalidPaths(t *testing.T) {
	directory := t.TempDir()
	filePath := filepath.Join(directory, "file")
	if err := os.WriteFile(filePath, []byte("not a directory"), 0o600); err != nil {
		t.Fatalf("create file: %v", err)
	}

	for _, path := range []string{filepath.Join(directory, "missing"), filePath} {
		if err := StorageDirectory(path)(context.Background()); err == nil {
			t.Fatalf("StorageDirectory(%q) error = nil, want invalid path error", path)
		}
	}
}
