package catalog

import (
	"context"
	"errors"
	"testing"
	"time"

	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
)

const (
	testItemID    = "11111111-1111-4111-8111-111111111111"
	testPartID    = "22222222-2222-4222-8222-222222222222"
	testVersionID = "33333333-3333-4333-8333-333333333333"
	testFileID    = "44444444-4444-4444-8444-444444444444"
	testActorID   = "55555555-5555-4555-8555-555555555555"
)

func TestDesignServiceNormalizesPartAndImmutableVersion(t *testing.T) {
	now := time.Date(2026, 9, 4, 15, 0, 0, 0, time.FixedZone("test", -3*60*60))
	repository := &designRepositoryStub{}
	service := &DesignService{repository: repository, now: func() time.Time { return now }}
	part, err := service.CreatePart(context.Background(), testItemID, domaincatalog.PartValues{Name: "  Base  ", Quantity: 2, Notes: "  print twice  "})
	if err != nil || repository.partValues.Name != "Base" || repository.partValues.Notes != "print twice" || !repository.at.Equal(now.UTC()) || part.Name != "Base" {
		t.Fatalf("CreatePart() = %#v, %v; values = %#v, at = %v", part, err, repository.partValues, repository.at)
	}
	allowed := false
	sourceURL := " https://example.com/design/42 "
	version, err := service.CreateVersion(context.Background(), testPartID, testActorID, domaincatalog.DesignVersionValues{
		Version: " v1 ", Origin: domaincatalog.DesignOriginThirdParty, SourceURL: &sourceURL,
		OriginalAuthor: " Maker ", LicenseName: " CC BY-NC ", CommercialUseAllowed: &allowed,
		AttributionRequired: true, AttributionText: " Maker, Example ",
	})
	if err != nil || repository.actorID != testActorID || repository.versionValues.Version != "v1" || *repository.versionValues.SourceURL != "https://example.com/design/42" || version.Version != "v1" {
		t.Fatalf("CreateVersion() = %#v, %v; actor = %q, values = %#v", version, err, repository.actorID, repository.versionValues)
	}
}

func TestDesignServiceRejectsInvalidData(t *testing.T) {
	service, err := NewDesignService(&designRepositoryStub{})
	if err != nil {
		t.Fatalf("NewDesignService() error = %v", err)
	}
	if _, err := service.CreatePart(context.Background(), testItemID, domaincatalog.PartValues{Name: "Part", Quantity: 0}); !errors.Is(err, ErrInvalidPart) {
		t.Fatalf("invalid part error = %v", err)
	}
	for _, values := range []domaincatalog.DesignVersionValues{
		{Version: "", Origin: domaincatalog.DesignOriginOriginal},
		{Version: "v1", Origin: "copied"},
		{Version: "v1", Origin: domaincatalog.DesignOriginThirdParty, AttributionRequired: true},
		{Version: "v1", Origin: domaincatalog.DesignOriginThirdParty, SourceURL: stringPointer("file:///private/design")},
	} {
		if _, err := service.CreateVersion(context.Background(), testPartID, testActorID, values); !errors.Is(err, ErrInvalidDesignVersion) {
			t.Fatalf("CreateVersion(%#v) error = %v", values, err)
		}
	}
	if _, err := service.AttachFile(context.Background(), testVersionID, testFileID, testActorID, "executable"); !errors.Is(err, ErrInvalidDesignFile) {
		t.Fatalf("invalid file role error = %v", err)
	}
}

func TestDesignServicePreservesRepositoryConflicts(t *testing.T) {
	repository := &designRepositoryStub{err: domaincatalog.ErrDesignVersionConflict}
	service, _ := NewDesignService(repository)
	_, err := service.CreateVersion(context.Background(), testPartID, testActorID, domaincatalog.DesignVersionValues{Version: "v1", Origin: domaincatalog.DesignOriginUnknown})
	if !errors.Is(err, domaincatalog.ErrDesignVersionConflict) {
		t.Fatalf("CreateVersion() error = %v", err)
	}
}

func stringPointer(value string) *string { return &value }

type designRepositoryStub struct {
	partValues    domaincatalog.PartValues
	versionValues domaincatalog.DesignVersionValues
	actorID       string
	at            time.Time
	err           error
}

func (stub *designRepositoryStub) CreatePart(_ context.Context, itemID string, values domaincatalog.PartValues, at time.Time) (domaincatalog.Part, error) {
	stub.partValues, stub.at = values, at
	return domaincatalog.Part{CatalogItemID: itemID, Name: values.Name, Quantity: values.Quantity, Notes: values.Notes}, stub.err
}
func (stub *designRepositoryStub) FindPart(context.Context, string) (domaincatalog.Part, error) {
	return domaincatalog.Part{}, stub.err
}
func (stub *designRepositoryStub) ListParts(context.Context, string) ([]domaincatalog.Part, error) {
	return []domaincatalog.Part{}, stub.err
}
func (stub *designRepositoryStub) UpdatePart(_ context.Context, _ string, values domaincatalog.PartValues, at time.Time) (domaincatalog.Part, error) {
	stub.partValues, stub.at = values, at
	return domaincatalog.Part{Name: values.Name}, stub.err
}
func (stub *designRepositoryStub) DeletePart(context.Context, string) error { return stub.err }
func (stub *designRepositoryStub) CreateDesignVersion(_ context.Context, partID, actorID string, values domaincatalog.DesignVersionValues, at time.Time) (domaincatalog.DesignVersion, error) {
	stub.versionValues, stub.actorID, stub.at = values, actorID, at
	return domaincatalog.DesignVersion{CatalogPartID: partID, Version: values.Version, Origin: values.Origin}, stub.err
}
func (stub *designRepositoryStub) ListDesignVersions(context.Context, string) ([]domaincatalog.DesignVersion, error) {
	return []domaincatalog.DesignVersion{}, stub.err
}
func (stub *designRepositoryStub) AttachDesignFile(context.Context, string, string, string, domaincatalog.DesignFileRole, time.Time) (domaincatalog.DesignFile, error) {
	return domaincatalog.DesignFile{}, stub.err
}
