package settings

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"
	"time"

	applicationstorage "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/storage"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
	domainsettings "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/settings"
)

const testLogoUploaderID = "11111111-1111-4111-8111-111111111111"

func TestLogoServiceValidatesStoresAndAssociatesPNG(t *testing.T) {
	now := time.Date(2026, time.September, 2, 18, 0, 0, 0, time.UTC)
	content := testPNG(t, 2, 2)
	store := &logoStoreStub{}
	repository := &logoRepositoryStub{file: domainfiles.File{ID: "file-id"}}
	service := newTestLogoService(t, repository, store, int64(len(content)+1), now)

	result, err := service.Upload(context.Background(), LogoUpload{
		UploadedBy:   testLogoUploaderID,
		OriginalName: "  workshop.png  ",
		Content:      bytes.NewReader(content),
	})
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if result.File.ID != "file-id" || repository.associateCalls != 1 {
		t.Fatalf("Upload() result = %#v, calls = %d", result, repository.associateCalls)
	}
	if repository.params.OriginalName != "workshop.png" || repository.params.ContentType != "image/png" ||
		repository.params.UploadedBy != testLogoUploaderID || repository.params.SizeBytes != int64(len(content)) ||
		!repository.updatedAt.Equal(now) {
		t.Fatalf("associated params = %#v at %s", repository.params, repository.updatedAt)
	}
	if !bytes.Equal(store.putContent, content) || len(repository.params.SHA256) != sha256.Size {
		t.Fatalf("stored bytes/hash = %d/%x", len(store.putContent), repository.params.SHA256)
	}
}

func TestLogoServiceRejectsInvalidUploadsBeforeStorage(t *testing.T) {
	valid := testPNG(t, 2, 2)
	tests := []struct {
		name    string
		upload  LogoUpload
		maximum int64
	}{
		{name: "missing reader", upload: LogoUpload{UploadedBy: testLogoUploaderID, OriginalName: "logo.png"}, maximum: 100},
		{name: "invalid uploader", upload: LogoUpload{UploadedBy: "not-a-uuid", OriginalName: "logo.png", Content: bytes.NewReader(valid)}, maximum: 1000},
		{name: "path filename", upload: LogoUpload{UploadedBy: testLogoUploaderID, OriginalName: `..\logo.png`, Content: bytes.NewReader(valid)}, maximum: 1000},
		{name: "not image", upload: LogoUpload{UploadedBy: testLogoUploaderID, OriginalName: "logo.png", Content: strings.NewReader("not an image")}, maximum: 1000},
		{name: "too large", upload: LogoUpload{UploadedBy: testLogoUploaderID, OriginalName: "logo.png", Content: bytes.NewReader(valid)}, maximum: int64(len(valid) - 1)},
		{name: "dimensions", upload: LogoUpload{UploadedBy: testLogoUploaderID, OriginalName: "logo.png", Content: bytes.NewReader(testPNG(t, maximumLogoDimension+1, 1))}, maximum: 1024 * 1024},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &logoStoreStub{}
			service := newTestLogoService(t, &logoRepositoryStub{}, store, test.maximum, time.Now())
			if _, err := service.Upload(context.Background(), test.upload); !errors.Is(err, ErrInvalidWorkshopLogo) {
				t.Fatalf("Upload() error = %v", err)
			}
			if store.putCalls != 0 {
				t.Fatalf("store calls = %d, want 0", store.putCalls)
			}
		})
	}
}

func TestValidateLogoImageAcceptsDeclaredFormats(t *testing.T) {
	var jpegBuffer bytes.Buffer
	if err := jpeg.Encode(&jpegBuffer, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatalf("encode JPEG: %v", err)
	}
	for contentType, content := range map[string][]byte{
		"image/png":  testPNG(t, 2, 2),
		"image/jpeg": jpegBuffer.Bytes(),
	} {
		got, err := validateLogoImage(content)
		if err != nil || got != contentType {
			t.Fatalf("validateLogoImage(%s) = %q, %v", contentType, got, err)
		}
	}
}

func TestLogoServiceOpensOnlyRepositoryAuthorizedCurrentLogo(t *testing.T) {
	key, err := applicationstorage.ParseObjectKey(strings.Repeat("a", 64))
	if err != nil {
		t.Fatalf("ParseObjectKey() error = %v", err)
	}
	repository := &logoRepositoryStub{current: domainfiles.File{ID: "current", StorageKey: key.String()}}
	store := &logoStoreStub{openContent: []byte("logo")}
	service := newTestLogoService(t, repository, store, 100, time.Now())

	download, err := service.OpenCurrent(context.Background())
	if err != nil {
		t.Fatalf("OpenCurrent() error = %v", err)
	}
	defer download.Content.Close()
	content, _ := io.ReadAll(download.Content)
	if download.File.ID != "current" || string(content) != "logo" || store.openKey.String() != key.String() {
		t.Fatalf("download = %#v, content = %q, key = %q", download.File, content, store.openKey.String())
	}
}

func TestNewLogoServiceRejectsInvalidConfiguration(t *testing.T) {
	store := &logoStoreStub{}
	repository := &logoRepositoryStub{}
	if _, err := NewLogoService(nil, store, 1); !errors.Is(err, ErrInvalidWorkshopLogoConfiguration) {
		t.Fatalf("nil repository error = %v", err)
	}
	if _, err := NewLogoService(repository, nil, 1); !errors.Is(err, ErrInvalidWorkshopLogoConfiguration) {
		t.Fatalf("nil store error = %v", err)
	}
	if _, err := NewLogoService(repository, store, 0); !errors.Is(err, ErrInvalidWorkshopLogoConfiguration) {
		t.Fatalf("zero maximum error = %v", err)
	}
}

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, width, height))
	picture.Set(0, 0, color.RGBA{R: 0x22, G: 0x66, B: 0xaa, A: 0xff})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, picture); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	return buffer.Bytes()
}

func newTestLogoService(t *testing.T, repository LogoRepository, store applicationstorage.Store, maximum int64, now time.Time) *LogoService {
	t.Helper()
	service, err := newLogoService(repository, store, maximum, func() time.Time { return now })
	if err != nil {
		t.Fatalf("newLogoService() error = %v", err)
	}
	return service
}

type logoRepositoryStub struct {
	file           domainfiles.File
	settings       domainsettings.WorkshopSettings
	associateError error
	params         domainfiles.CreateParams
	updatedAt      time.Time
	associateCalls int
	current        domainfiles.File
	currentError   error
}

func (stub *logoRepositoryStub) AssociateLogo(
	_ context.Context,
	params domainfiles.CreateParams,
	updatedAt time.Time,
) (domainfiles.File, domainsettings.WorkshopSettings, error) {
	stub.associateCalls++
	stub.params = params
	stub.updatedAt = updatedAt
	return stub.file, stub.settings, stub.associateError
}

func (stub *logoRepositoryStub) CurrentLogo(context.Context) (domainfiles.File, error) {
	return stub.current, stub.currentError
}

type logoStoreStub struct {
	putContent  []byte
	putCalls    int
	putError    error
	openContent []byte
	openKey     applicationstorage.ObjectKey
	openError   error
}

func (stub *logoStoreStub) Put(_ context.Context, source io.Reader) (applicationstorage.StoredObject, error) {
	stub.putCalls++
	stub.putContent, _ = io.ReadAll(source)
	if stub.putError != nil {
		return applicationstorage.StoredObject{}, stub.putError
	}
	digest := applicationstorage.SHA256Digest(sha256.Sum256(stub.putContent))
	key, _ := applicationstorage.ParseObjectKey(digest.String())
	return applicationstorage.StoredObject{Key: key, SHA256: digest, SizeBytes: int64(len(stub.putContent))}, nil
}

func (stub *logoStoreStub) Open(_ context.Context, key applicationstorage.ObjectKey) (io.ReadCloser, error) {
	stub.openKey = key
	if stub.openError != nil {
		return nil, stub.openError
	}
	return io.NopCloser(bytes.NewReader(stub.openContent)), nil
}

func (*logoStoreStub) DeleteForCleanup(context.Context, applicationstorage.ObjectKey) error {
	return nil
}
