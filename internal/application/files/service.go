// Package files implements authorized immutable file transfer workflows.
package files

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"regexp"
	"strings"
	"unicode/utf8"

	applicationstorage "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/storage"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
)

const (
	maximumOriginalNameRunes = 255
	maximumContentTypeRunes  = 200
)

var (
	ErrInvalidUpload        = errors.New("invalid file upload")
	ErrUploadTooLarge       = errors.New("file upload too large")
	ErrInvalidFileID        = errors.New("invalid file id")
	ErrInvalidConfiguration = errors.New("invalid file service configuration")
	uuidPattern             = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// Repository persists immutable metadata and resolves downloads.
type Repository interface {
	CreateOrGet(context.Context, domainfiles.CreateParams) (domainfiles.File, bool, error)
	FindByID(context.Context, string) (domainfiles.File, error)
}

// Upload is untrusted multipart metadata and content from an authenticated user.
type Upload struct {
	UploadedBy   string
	OriginalName string
	ContentType  string
	Content      io.Reader
}

// UploadResult reports the canonical metadata and content deduplication outcome.
type UploadResult struct {
	File         domainfiles.File
	Deduplicated bool
}

// Download owns a storage reader that the HTTP handler must close.
type Download struct {
	File    domainfiles.File
	Content io.ReadCloser
}

// Service coordinates immutable storage and metadata persistence.
type Service struct {
	repository   Repository
	store        applicationstorage.Store
	maximumBytes int64
}

// NewService validates file transfer dependencies and policy.
func NewService(repository Repository, store applicationstorage.Store, maximumBytes int64) (*Service, error) {
	if repository == nil || store == nil || maximumBytes <= 0 {
		return nil, ErrInvalidConfiguration
	}
	return &Service{repository: repository, store: store, maximumBytes: maximumBytes}, nil
}

// UploadFile streams a bounded object, then creates or reuses its SHA-256 metadata.
func (service *Service) UploadFile(ctx context.Context, upload Upload) (UploadResult, error) {
	name := strings.TrimSpace(upload.OriginalName)
	contentType, err := normalizeContentType(upload.ContentType)
	if !validUUID(upload.UploadedBy) || !validOriginalName(name) || err != nil || upload.Content == nil {
		return UploadResult{}, ErrInvalidUpload
	}
	stored, err := service.store.Put(ctx, io.LimitReader(upload.Content, service.maximumBytes+1))
	if err != nil {
		return UploadResult{}, fmt.Errorf("store uploaded file: %w", err)
	}
	if stored.SizeBytes == 0 {
		return UploadResult{}, service.cleanup(ctx, stored, ErrInvalidUpload)
	}
	if stored.SizeBytes > service.maximumBytes {
		return UploadResult{}, service.cleanup(ctx, stored, ErrUploadTooLarge)
	}
	file, created, err := service.repository.CreateOrGet(ctx, domainfiles.CreateParams{
		SHA256:       append([]byte(nil), stored.SHA256[:]...),
		OriginalName: name,
		ContentType:  contentType,
		SizeBytes:    stored.SizeBytes,
		StorageKey:   stored.Key.String(),
		UploadedBy:   upload.UploadedBy,
	})
	if err != nil {
		return UploadResult{}, service.cleanup(ctx, stored, fmt.Errorf("persist uploaded file: %w", err))
	}
	return UploadResult{File: file, Deduplicated: !created}, nil
}

// OpenFile validates the public identifier and opens its authorized object.
func (service *Service) OpenFile(ctx context.Context, id string) (Download, error) {
	normalizedID := strings.ToLower(strings.TrimSpace(id))
	if !validUUID(normalizedID) {
		return Download{}, ErrInvalidFileID
	}
	file, err := service.repository.FindByID(ctx, normalizedID)
	if err != nil {
		return Download{}, err
	}
	key, err := applicationstorage.ParseObjectKey(file.StorageKey)
	if err != nil {
		return Download{}, fmt.Errorf("parse file storage key: %w", err)
	}
	content, err := service.store.Open(ctx, key)
	if err != nil {
		return Download{}, fmt.Errorf("open file object: %w", err)
	}
	return Download{File: file, Content: content}, nil
}

func (service *Service) cleanup(ctx context.Context, stored applicationstorage.StoredObject, cause error) error {
	if !stored.Created {
		return cause
	}
	if err := service.store.DeleteForCleanup(context.WithoutCancel(ctx), stored.Key); err != nil {
		return errors.Join(cause, fmt.Errorf("clean up uploaded file: %w", err))
	}
	return cause
}

func normalizeContentType(value string) (string, error) {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > maximumContentTypeRunes {
		return "", ErrInvalidUpload
	}
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err != nil || mediaType == "" {
		return "", ErrInvalidUpload
	}
	return strings.ToLower(mediaType), nil
}

func validOriginalName(name string) bool {
	if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > maximumOriginalNameRunes ||
		strings.ContainsAny(name, `/\`) {
		return false
	}
	for _, character := range name {
		if character < 0x20 || character == 0x7f {
			return false
		}
	}
	return true
}

func validUUID(value string) bool {
	return uuidPattern.MatchString(strings.ToLower(strings.TrimSpace(value)))
}
