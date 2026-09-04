package files

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"strings"
	"testing"

	applicationstorage "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/storage"
	domainfiles "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/files"
)

const testUploaderID = "11111111-1111-4111-8111-111111111111"

func TestServiceUploadsAndDeduplicatesImmutableFile(t *testing.T) {
	content := []byte("solid mesh")
	repository := &repositoryStub{file: domainfiles.File{ID: "file-id"}, created: true}
	store := newStoreStub(content, true)
	service, _ := NewService(repository, store, 100)
	result, err := service.UploadFile(context.Background(), Upload{UploadedBy: testUploaderID, OriginalName: " part.stl ", ContentType: "model/stl; charset=binary", Content: bytes.NewReader(content)})
	if err != nil || result.File.ID != "file-id" || result.Deduplicated {
		t.Fatalf("UploadFile() = %#v, %v", result, err)
	}
	if repository.params.OriginalName != "part.stl" || repository.params.ContentType != "model/stl" || repository.params.SizeBytes != int64(len(content)) || len(repository.params.SHA256) != sha256.Size {
		t.Fatalf("repository params = %#v", repository.params)
	}

	repository.created = false
	store.stored.Created = false
	result, err = service.UploadFile(context.Background(), Upload{UploadedBy: testUploaderID, OriginalName: "copy.stl", ContentType: "model/stl", Content: bytes.NewReader(content)})
	if err != nil || !result.Deduplicated || store.deleteCalls != 0 {
		t.Fatalf("deduplicated UploadFile() = %#v, %v; deletes %d", result, err, store.deleteCalls)
	}
}

func TestServiceRejectsInvalidAndOversizedUploadsWithSafeCleanup(t *testing.T) {
	tests := []Upload{
		{UploadedBy: "bad", OriginalName: "part.stl", ContentType: "model/stl", Content: strings.NewReader("x")},
		{UploadedBy: testUploaderID, OriginalName: `..\part.stl`, ContentType: "model/stl", Content: strings.NewReader("x")},
		{UploadedBy: testUploaderID, OriginalName: "part.stl", ContentType: "bad type", Content: strings.NewReader("x")},
	}
	for _, upload := range tests {
		store := newStoreStub([]byte("x"), true)
		service, _ := NewService(&repositoryStub{}, store, 10)
		if _, err := service.UploadFile(context.Background(), upload); !errors.Is(err, ErrInvalidUpload) || store.putCalls != 0 {
			t.Fatalf("UploadFile(%#v) error = %v, puts = %d", upload, err, store.putCalls)
		}
	}

	store := newStoreStub([]byte("123456"), true)
	service, _ := NewService(&repositoryStub{}, store, 5)
	if _, err := service.UploadFile(context.Background(), Upload{UploadedBy: testUploaderID, OriginalName: "large.bin", ContentType: "application/octet-stream", Content: strings.NewReader("123456789")}); !errors.Is(err, ErrUploadTooLarge) || store.deleteCalls != 1 {
		t.Fatalf("oversized UploadFile() error = %v, deletes = %d", err, store.deleteCalls)
	}
}

func TestServiceCleansNewObjectOnMetadataFailureButPreservesDeduplicatedObject(t *testing.T) {
	for _, created := range []bool{true, false} {
		store := newStoreStub([]byte("content"), created)
		service, _ := NewService(&repositoryStub{err: errors.New("database unavailable")}, store, 100)
		_, err := service.UploadFile(context.Background(), Upload{UploadedBy: testUploaderID, OriginalName: "file.bin", ContentType: "application/octet-stream", Content: strings.NewReader("content")})
		if err == nil || store.deleteCalls != boolInt(created) {
			t.Fatalf("created %t: error = %v, deletes = %d", created, err, store.deleteCalls)
		}
	}
}

func TestServiceOpensFileByValidatedID(t *testing.T) {
	key, _ := applicationstorage.ParseObjectKey(strings.Repeat("a", 64))
	repository := &repositoryStub{file: domainfiles.File{ID: testUploaderID, StorageKey: key.String()}}
	store := newStoreStub([]byte("download"), false)
	service, _ := NewService(repository, store, 100)
	download, err := service.OpenFile(context.Background(), testUploaderID)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer download.Content.Close()
	content, _ := io.ReadAll(download.Content)
	if string(content) != "download" || store.openKey.String() != key.String() {
		t.Fatalf("OpenFile() content = %q, key = %q", content, store.openKey.String())
	}
	if _, err := service.OpenFile(context.Background(), "not-a-uuid"); !errors.Is(err, ErrInvalidFileID) {
		t.Fatalf("OpenFile() invalid error = %v", err)
	}
}

type repositoryStub struct {
	file    domainfiles.File
	created bool
	err     error
	params  domainfiles.CreateParams
}

func (stub *repositoryStub) CreateOrGet(_ context.Context, params domainfiles.CreateParams) (domainfiles.File, bool, error) {
	stub.params = params
	return stub.file, stub.created, stub.err
}

func (stub *repositoryStub) FindByID(context.Context, string) (domainfiles.File, error) {
	return stub.file, stub.err
}

type storeStub struct {
	stored      applicationstorage.StoredObject
	content     []byte
	putCalls    int
	deleteCalls int
	openKey     applicationstorage.ObjectKey
}

func newStoreStub(content []byte, created bool) *storeStub {
	digest := applicationstorage.SHA256Digest(sha256.Sum256(content))
	key, _ := applicationstorage.ParseObjectKey(digest.String())
	return &storeStub{stored: applicationstorage.StoredObject{Key: key, SHA256: digest, SizeBytes: int64(len(content)), Created: created}, content: content}
}

func (stub *storeStub) Put(context.Context, io.Reader) (applicationstorage.StoredObject, error) {
	stub.putCalls++
	return stub.stored, nil
}

func (stub *storeStub) Open(_ context.Context, key applicationstorage.ObjectKey) (io.ReadCloser, error) {
	stub.openKey = key
	return io.NopCloser(bytes.NewReader(stub.content)), nil
}

func (stub *storeStub) DeleteForCleanup(context.Context, applicationstorage.ObjectKey) error {
	stub.deleteCalls++
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
