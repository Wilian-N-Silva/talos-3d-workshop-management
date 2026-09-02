// Package files contains immutable file metadata definitions.
package files

import (
	"errors"
	"time"
)

// ErrFileNotFound indicates that no authorized file record matches a lookup.
var ErrFileNotFound = errors.New("file not found")

// File is immutable metadata for one server-side object.
type File struct {
	ID           string
	SHA256       []byte
	OriginalName string
	ContentType  string
	SizeBytes    int64
	StorageKey   string
	UploadedBy   string
	CreatedAt    time.Time
}

// CreateParams contains validated metadata returned by object storage.
type CreateParams struct {
	SHA256       []byte
	OriginalName string
	ContentType  string
	SizeBytes    int64
	StorageKey   string
	UploadedBy   string
}
