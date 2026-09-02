package settings

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	applicationstorage "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/storage"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
)

const (
	DefaultMaximumLogoBytes  = 5 * 1024 * 1024
	maximumLogoDimension     = 4096
	maximumOriginalNameRunes = 255
)

var (
	// ErrInvalidWorkshopLogo indicates malformed metadata or unsupported image bytes.
	ErrInvalidWorkshopLogo = errors.New("invalid workshop logo")
	// ErrInvalidWorkshopLogoConfiguration indicates missing dependencies or an invalid size policy.
	ErrInvalidWorkshopLogoConfiguration = errors.New("invalid workshop logo configuration")
	logoUploaderIDPattern               = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// LogoRepository atomically records immutable file metadata and its workshop association.
type LogoRepository interface {
	AssociateLogo(context.Context, domainfiles.CreateParams, time.Time) (domainfiles.File, domainsettings.WorkshopSettings, error)
	CurrentLogo(context.Context) (domainfiles.File, error)
}

// LogoUpload contains trusted upload metadata and untrusted image bytes.
type LogoUpload struct {
	UploadedBy   string
	OriginalName string
	Content      io.Reader
}

// LogoUploadResult is the associated immutable file and updated settings record.
type LogoUploadResult struct {
	File     domainfiles.File
	Settings domainsettings.WorkshopSettings
}

// LogoDownload owns an open object reader that the caller must close.
type LogoDownload struct {
	File    domainfiles.File
	Content io.ReadCloser
}

// LogoService validates, stores, associates, and opens the current workshop logo.
type LogoService struct {
	repository   LogoRepository
	store        applicationstorage.Store
	maximumBytes int64
	now          func() time.Time
}

// NewLogoService creates the workshop-logo application boundary.
func NewLogoService(
	repository LogoRepository,
	store applicationstorage.Store,
	maximumBytes int64,
) (*LogoService, error) {
	return newLogoService(repository, store, maximumBytes, time.Now)
}

func newLogoService(
	repository LogoRepository,
	store applicationstorage.Store,
	maximumBytes int64,
	now func() time.Time,
) (*LogoService, error) {
	if repository == nil || store == nil || maximumBytes <= 0 || now == nil {
		return nil, ErrInvalidWorkshopLogoConfiguration
	}
	return &LogoService{repository: repository, store: store, maximumBytes: maximumBytes, now: now}, nil
}

// Upload validates the complete PNG/JPEG before publishing it to immutable storage.
func (service *LogoService) Upload(ctx context.Context, upload LogoUpload) (LogoUploadResult, error) {
	name := strings.TrimSpace(upload.OriginalName)
	if !validOriginalName(name) || !logoUploaderIDPattern.MatchString(upload.UploadedBy) || upload.Content == nil {
		return LogoUploadResult{}, ErrInvalidWorkshopLogo
	}

	content, err := io.ReadAll(io.LimitReader(upload.Content, service.maximumBytes+1))
	if err != nil {
		return LogoUploadResult{}, fmt.Errorf("read workshop logo: %w", err)
	}
	if int64(len(content)) == 0 || int64(len(content)) > service.maximumBytes {
		return LogoUploadResult{}, ErrInvalidWorkshopLogo
	}
	contentType, err := validateLogoImage(content)
	if err != nil {
		return LogoUploadResult{}, err
	}

	stored, err := service.store.Put(ctx, bytes.NewReader(content))
	if err != nil {
		return LogoUploadResult{}, fmt.Errorf("store workshop logo: %w", err)
	}
	file, settings, err := service.repository.AssociateLogo(ctx, domainfiles.CreateParams{
		SHA256:       append([]byte(nil), stored.SHA256[:]...),
		OriginalName: name,
		ContentType:  contentType,
		SizeBytes:    stored.SizeBytes,
		StorageKey:   stored.Key.String(),
		UploadedBy:   upload.UploadedBy,
	}, service.now().UTC())
	if err != nil {
		return LogoUploadResult{}, fmt.Errorf("associate workshop logo: %w", err)
	}
	return LogoUploadResult{File: file, Settings: settings}, nil
}

// OpenCurrent opens only the file currently authorized as public workshop branding.
func (service *LogoService) OpenCurrent(ctx context.Context) (LogoDownload, error) {
	file, err := service.repository.CurrentLogo(ctx)
	if err != nil {
		return LogoDownload{}, fmt.Errorf("get current workshop logo: %w", err)
	}
	key, err := applicationstorage.ParseObjectKey(file.StorageKey)
	if err != nil {
		return LogoDownload{}, fmt.Errorf("parse workshop logo storage key: %w", err)
	}
	content, err := service.store.Open(ctx, key)
	if err != nil {
		return LogoDownload{}, fmt.Errorf("open workshop logo object: %w", err)
	}
	return LogoDownload{File: file, Content: content}, nil
}

func validateLogoImage(content []byte) (string, error) {
	configuration, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || configuration.Width < 1 || configuration.Height < 1 ||
		configuration.Width > maximumLogoDimension || configuration.Height > maximumLogoDimension {
		return "", ErrInvalidWorkshopLogo
	}
	var contentType string
	switch format {
	case "png":
		contentType = "image/png"
	case "jpeg":
		contentType = "image/jpeg"
	default:
		return "", ErrInvalidWorkshopLogo
	}
	if _, decodedFormat, err := image.Decode(bytes.NewReader(content)); err != nil || decodedFormat != format {
		return "", ErrInvalidWorkshopLogo
	}
	return contentType, nil
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
