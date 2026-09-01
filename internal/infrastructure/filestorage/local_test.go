package filestorage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	applicationstorage "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/storage"
)

var errInterruptedRead = errors.New("interrupted read")

type interruptedReader struct {
	sent bool
}

func (reader *interruptedReader) Read(buffer []byte) (int, error) {
	if reader.sent {
		return 0, errInterruptedRead
	}
	reader.sent = true
	return copy(buffer, []byte("partial content")), nil
}

func TestLocalFilesystemStoragePutAndOpen(t *testing.T) {
	dataDirectory := t.TempDir()
	store := newTestStore(t, dataDirectory)
	content := []byte("immutable workshop object")

	stored, err := store.Put(context.Background(), bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	expectedDigest := applicationstorage.SHA256Digest(sha256.Sum256(content))
	if stored.SHA256 != expectedDigest {
		t.Fatalf("SHA256 = %s, want %s", stored.SHA256.String(), expectedDigest.String())
	}
	if stored.Key.String() != expectedDigest.String() {
		t.Fatalf("key = %q, want content digest", stored.Key.String())
	}
	if stored.SizeBytes != int64(len(content)) {
		t.Fatalf("SizeBytes = %d, want %d", stored.SizeBytes, len(content))
	}

	expectedPath := filepath.Join(dataDirectory, "objects", stored.Key.String()[:2], stored.Key.String())
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("stat content-addressed object: %v", err)
	}

	reader, err := store.Open(context.Background(), stored.Key)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	opened, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close object: %v", err)
	}
	if !bytes.Equal(opened, content) {
		t.Fatalf("opened content = %q, want %q", opened, content)
	}

	assertNoStagedObjects(t, store)
}

func TestLocalFilesystemStorageDeduplicatesConcurrentWrites(t *testing.T) {
	store := newTestStore(t, t.TempDir())
	content := []byte("same immutable bytes")

	const writers = 8
	results := make(chan applicationstorage.StoredObject, writers)
	errorsChannel := make(chan error, writers)
	var waitGroup sync.WaitGroup
	for range writers {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			stored, err := store.Put(context.Background(), bytes.NewReader(content))
			if err != nil {
				errorsChannel <- err
				return
			}
			results <- stored
		}()
	}
	waitGroup.Wait()
	close(results)
	close(errorsChannel)

	for err := range errorsChannel {
		t.Errorf("concurrent Put() error = %v", err)
	}
	var expectedKey string
	resultCount := 0
	for stored := range results {
		resultCount++
		if expectedKey == "" {
			expectedKey = stored.Key.String()
		}
		if stored.Key.String() != expectedKey {
			t.Errorf("key = %q, want %q", stored.Key.String(), expectedKey)
		}
	}
	if resultCount != writers {
		t.Fatalf("successful Put() results = %d, want %d", resultCount, writers)
	}

	shardEntries, err := os.ReadDir(filepath.Join(store.objectsDirectory, expectedKey[:2]))
	if err != nil {
		t.Fatalf("read shard: %v", err)
	}
	if len(shardEntries) != 1 || shardEntries[0].Name() != expectedKey {
		t.Fatalf("shard entries = %v, want one deduplicated object", shardEntries)
	}
	assertNoStagedObjects(t, store)
}

func TestLocalFilesystemStorageDoesNotPublishInterruptedWrites(t *testing.T) {
	store := newTestStore(t, t.TempDir())

	if _, err := store.Put(context.Background(), &interruptedReader{}); !errors.Is(err, errInterruptedRead) {
		t.Fatalf("Put() error = %v, want interrupted read error", err)
	}
	entries, err := os.ReadDir(store.objectsDirectory)
	if err != nil {
		t.Fatalf("read objects directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != temporaryDirectory {
		t.Fatalf("object entries = %v, want only staging directory", entries)
	}
	assertNoStagedObjects(t, store)
}

func TestLocalFilesystemStorageNeverOverwritesExistingObject(t *testing.T) {
	store := newTestStore(t, t.TempDir())
	content := []byte("expected content")
	stored, err := store.Put(context.Background(), bytes.NewReader(content))
	if err != nil {
		t.Fatalf("initial Put() error = %v", err)
	}
	objectPath, err := store.objectPath(stored.Key)
	if err != nil {
		t.Fatalf("objectPath() error = %v", err)
	}
	corruptContent := []byte("externally corrupted")
	if err := os.WriteFile(objectPath, corruptContent, 0o600); err != nil {
		t.Fatalf("corrupt stored object: %v", err)
	}

	if _, err := store.Put(context.Background(), bytes.NewReader(content)); !errors.Is(err, ErrObjectIntegrity) {
		t.Fatalf("second Put() error = %v, want ErrObjectIntegrity", err)
	}
	actual, err := os.ReadFile(objectPath)
	if err != nil {
		t.Fatalf("read stored object: %v", err)
	}
	if !bytes.Equal(actual, corruptContent) {
		t.Fatalf("existing object was overwritten: got %q", actual)
	}
}

func TestLocalFilesystemStorageDeleteForCleanupIsExactAndIdempotent(t *testing.T) {
	store := newTestStore(t, t.TempDir())
	first, err := store.Put(context.Background(), bytes.NewReader([]byte("first")))
	if err != nil {
		t.Fatalf("put first object: %v", err)
	}
	second, err := store.Put(context.Background(), bytes.NewReader([]byte("second")))
	if err != nil {
		t.Fatalf("put second object: %v", err)
	}

	if err := store.DeleteForCleanup(context.Background(), first.Key); err != nil {
		t.Fatalf("DeleteForCleanup() error = %v", err)
	}
	if err := store.DeleteForCleanup(context.Background(), first.Key); err != nil {
		t.Fatalf("idempotent DeleteForCleanup() error = %v", err)
	}
	if _, err := store.Open(context.Background(), first.Key); !errors.Is(err, applicationstorage.ErrObjectNotFound) {
		t.Fatalf("Open(deleted) error = %v, want ErrObjectNotFound", err)
	}
	reader, err := store.Open(context.Background(), second.Key)
	if err != nil {
		t.Fatalf("Open(second) error = %v", err)
	}
	_ = reader.Close()
}

func TestLocalFilesystemStorageRejectsInvalidKeys(t *testing.T) {
	store := newTestStore(t, t.TempDir())
	var invalidKey applicationstorage.ObjectKey

	if _, err := store.Open(context.Background(), invalidKey); !errors.Is(err, applicationstorage.ErrInvalidObjectKey) {
		t.Fatalf("Open() error = %v, want ErrInvalidObjectKey", err)
	}
	if err := store.DeleteForCleanup(context.Background(), invalidKey); !errors.Is(err, applicationstorage.ErrInvalidObjectKey) {
		t.Fatalf("DeleteForCleanup() error = %v, want ErrInvalidObjectKey", err)
	}
}

func TestLocalFilesystemStorageHonorsCancelledContext(t *testing.T) {
	store := newTestStore(t, t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := store.Put(ctx, bytes.NewReader([]byte("not stored"))); !errors.Is(err, context.Canceled) {
		t.Fatalf("Put() error = %v, want context canceled", err)
	}
	assertNoStagedObjects(t, store)
}

func TestNewLocalFilesystemStorageRejectsFilePath(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(filePath, []byte("file"), 0o600); err != nil {
		t.Fatalf("create file: %v", err)
	}

	if _, err := NewLocalFilesystemStorage(filePath); err == nil {
		t.Fatal("NewLocalFilesystemStorage() error = nil, want file path error")
	}
}

func newTestStore(t *testing.T, dataDirectory string) *LocalFilesystemStorage {
	t.Helper()
	store, err := NewLocalFilesystemStorage(dataDirectory)
	if err != nil {
		t.Fatalf("NewLocalFilesystemStorage() error = %v", err)
	}
	return store
}

func assertNoStagedObjects(t *testing.T, store *LocalFilesystemStorage) {
	t.Helper()
	entries, err := os.ReadDir(store.temporaryDirectory)
	if err != nil {
		t.Fatalf("read staging directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("staging entries = %v, want none", entries)
	}
}
