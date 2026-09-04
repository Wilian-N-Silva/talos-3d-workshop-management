// Package filestorage contains server-side object storage adapters.
package filestorage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	applicationstorage "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/storage"
)

const (
	objectsDirectoryName = "objects"
	temporaryDirectory   = ".tmp"
)

// ErrObjectIntegrity indicates that existing bytes do not match their
// content-addressed key.
var ErrObjectIntegrity = errors.New("stored object failed integrity check")

// LocalFilesystemStorage stores immutable content-addressed objects below a
// server-controlled data directory.
type LocalFilesystemStorage struct {
	objectsDirectory   string
	temporaryDirectory string
}

var _ applicationstorage.Store = (*LocalFilesystemStorage)(nil)

// NewLocalFilesystemStorage initializes the object and staging directories.
func NewLocalFilesystemStorage(dataDirectory string) (*LocalFilesystemStorage, error) {
	if strings.TrimSpace(dataDirectory) == "" {
		return nil, fmt.Errorf("initialize local file storage: data directory is required")
	}

	absoluteDataDirectory, err := filepath.Abs(dataDirectory)
	if err != nil {
		return nil, fmt.Errorf("resolve data directory: %w", err)
	}

	objectsDirectory := filepath.Join(absoluteDataDirectory, objectsDirectoryName)
	tempDirectory := filepath.Join(objectsDirectory, temporaryDirectory)
	for _, directory := range []string{objectsDirectory, tempDirectory} {
		if err := os.MkdirAll(directory, 0o750); err != nil {
			return nil, fmt.Errorf("create local file storage directory: %w", err)
		}
		info, err := os.Stat(directory)
		if err != nil {
			return nil, fmt.Errorf("inspect local file storage directory: %w", err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("initialize local file storage: path is not a directory")
		}
	}

	return &LocalFilesystemStorage{
		objectsDirectory:   objectsDirectory,
		temporaryDirectory: tempDirectory,
	}, nil
}

// Put writes source to a staging file, computes its SHA-256 digest, and
// atomically publishes the object without replacing an existing path.
func (storage *LocalFilesystemStorage) Put(
	ctx context.Context,
	source io.Reader,
) (applicationstorage.StoredObject, error) {
	if err := ctx.Err(); err != nil {
		return applicationstorage.StoredObject{}, err
	}
	if source == nil {
		return applicationstorage.StoredObject{}, fmt.Errorf("store object: source is required")
	}

	staged, err := os.CreateTemp(storage.temporaryDirectory, ".object-*")
	if err != nil {
		return applicationstorage.StoredObject{}, fmt.Errorf("create staged object: %w", err)
	}
	stagedPath := staged.Name()
	defer func() {
		_ = staged.Close()
		_ = os.Remove(stagedPath)
	}()

	hasher := sha256.New()
	sizeBytes, err := io.Copy(io.MultiWriter(staged, hasher), readerWithContext{ctx: ctx, source: source})
	if err != nil {
		return applicationstorage.StoredObject{}, fmt.Errorf("write staged object: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return applicationstorage.StoredObject{}, err
	}
	if err := staged.Sync(); err != nil {
		return applicationstorage.StoredObject{}, fmt.Errorf("sync staged object: %w", err)
	}
	if err := staged.Close(); err != nil {
		return applicationstorage.StoredObject{}, fmt.Errorf("close staged object: %w", err)
	}

	var digest applicationstorage.SHA256Digest
	copy(digest[:], hasher.Sum(nil))
	key, err := applicationstorage.ParseObjectKey(digest.String())
	if err != nil {
		return applicationstorage.StoredObject{}, fmt.Errorf("create content-addressed key: %w", err)
	}
	storedObject := applicationstorage.StoredObject{
		Key:       key,
		SHA256:    digest,
		SizeBytes: sizeBytes,
	}

	targetPath, err := storage.ensureObjectPath(key)
	if err != nil {
		return applicationstorage.StoredObject{}, err
	}
	if err := ctx.Err(); err != nil {
		return applicationstorage.StoredObject{}, err
	}

	if err := os.Link(stagedPath, targetPath); err == nil {
		storedObject.Created = true
		return storedObject, nil
	} else if !errors.Is(err, fs.ErrExist) {
		return applicationstorage.StoredObject{}, fmt.Errorf("publish stored object: %w", err)
	}

	if err := verifyExistingObject(ctx, targetPath, digest, sizeBytes); err != nil {
		return applicationstorage.StoredObject{}, err
	}

	return storedObject, nil
}

// Open opens an immutable object for reading.
func (storage *LocalFilesystemStorage) Open(
	ctx context.Context,
	key applicationstorage.ObjectKey,
) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	objectPath, err := storage.objectPath(key)
	if err != nil {
		return nil, err
	}
	object, err := os.Open(objectPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, applicationstorage.ErrObjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open stored object: %w", err)
	}

	return object, nil
}

// DeleteForCleanup removes only the exact object represented by key. Missing
// objects are treated as an already-completed cleanup.
func (storage *LocalFilesystemStorage) DeleteForCleanup(
	ctx context.Context,
	key applicationstorage.ObjectKey,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	objectPath, err := storage.objectPath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(objectPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("clean up stored object: %w", err)
	}

	return nil
}

func (storage *LocalFilesystemStorage) ensureObjectPath(key applicationstorage.ObjectKey) (string, error) {
	objectPath, err := storage.objectPath(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(objectPath), 0o750); err != nil {
		return "", fmt.Errorf("create object shard directory: %w", err)
	}

	return objectPath, nil
}

func (storage *LocalFilesystemStorage) objectPath(key applicationstorage.ObjectKey) (string, error) {
	if !key.Valid() {
		return "", applicationstorage.ErrInvalidObjectKey
	}

	serializedKey := key.String()
	shard := serializedKey
	if len(shard) > 2 {
		shard = shard[:2]
	}

	return filepath.Join(storage.objectsDirectory, shard, serializedKey), nil
}

func verifyExistingObject(
	ctx context.Context,
	path string,
	expectedDigest applicationstorage.SHA256Digest,
	expectedSize int64,
) error {
	object, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%w: open existing object", ErrObjectIntegrity)
	}
	defer object.Close()

	hasher := sha256.New()
	sizeBytes, err := io.Copy(hasher, readerWithContext{ctx: ctx, source: object})
	if err != nil {
		if contextError := ctx.Err(); contextError != nil {
			return contextError
		}
		return fmt.Errorf("%w: read existing object", ErrObjectIntegrity)
	}
	if sizeBytes != expectedSize || !bytes.Equal(hasher.Sum(nil), expectedDigest[:]) {
		return ErrObjectIntegrity
	}

	return nil
}

type readerWithContext struct {
	ctx    context.Context
	source io.Reader
}

func (reader readerWithContext) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}

	return reader.source.Read(buffer)
}
