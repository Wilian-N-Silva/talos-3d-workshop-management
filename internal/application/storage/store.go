// Package storage defines the application boundary for immutable server-side
// object storage.
package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
)

// ErrObjectNotFound is returned when an object key does not exist.
var ErrObjectNotFound = errors.New("stored object not found")

// SHA256Digest is the binary SHA-256 digest of stored object content.
type SHA256Digest [sha256.Size]byte

// String returns the canonical lowercase hexadecimal digest.
func (digest SHA256Digest) String() string {
	return hex.EncodeToString(digest[:])
}

// StoredObject describes immutable content accepted by a Store. Original
// filenames, content types, owners, and catalog associations are metadata
// concerns and intentionally do not cross this boundary.
type StoredObject struct {
	Key       ObjectKey
	SHA256    SHA256Digest
	SizeBytes int64
	// Created is true only when this Put published the object. Callers use it
	// to avoid deleting a pre-existing deduplicated object during compensation.
	Created bool
}

// Store persists and opens immutable objects.
type Store interface {
	// Put consumes source without closing it and returns the resulting object.
	// Implementations may deduplicate identical content but must never overwrite
	// bytes belonging to an existing object key.
	Put(ctx context.Context, source io.Reader) (StoredObject, error)

	// Open returns a new reader for an existing object. The caller must close it.
	Open(ctx context.Context, key ObjectKey) (io.ReadCloser, error)

	// DeleteForCleanup removes an unreferenced object only to compensate for a
	// failed creation workflow. It is not a user-facing object deletion operation
	// and must be safe to retry when the object is already absent.
	DeleteForCleanup(ctx context.Context, key ObjectKey) error
}
