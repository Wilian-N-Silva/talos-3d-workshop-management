package catalog

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
)

func TestCreateNormalizesCatalogValues(t *testing.T) {
	now := time.Date(2026, 9, 4, 1, 2, 3, 0, time.FixedZone("test", -3*60*60))
	repository := &catalogRepositoryStub{}
	service, err := newService(repository, func() time.Time { return now })
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}
	emptySKU := "  "

	created, err := service.Create(context.Background(), domaincatalog.Values{
		Name: "  Calibration Cube  ", SKU: &emptySKU, Description: "  Small test  ",
		Purpose: domaincatalog.PurposeTest, Tags: []string{" Calibration ", "PLA", "pla"},
		Status: domaincatalog.StatusActive,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repository.created.Name != "Calibration Cube" || repository.created.SKU != nil || repository.created.Description != "Small test" {
		t.Fatalf("normalized values = %#v", repository.created)
	}
	if !reflect.DeepEqual(repository.created.Tags, []string{"calibration", "pla"}) {
		t.Fatalf("tags = %#v", repository.created.Tags)
	}
	if !repository.createdAt.Equal(now.UTC()) || created.Name != "Calibration Cube" {
		t.Fatalf("created at/item = %v, %#v", repository.createdAt, created)
	}
}

func TestCreateRejectsInvalidCatalogValues(t *testing.T) {
	service, err := NewService(&catalogRepositoryStub{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	valid := domaincatalog.Values{Name: "Part", Purpose: domaincatalog.PurposeProduct, Status: domaincatalog.StatusActive}
	tests := []struct {
		name   string
		mutate func(*domaincatalog.Values)
	}{
		{name: "empty name", mutate: func(value *domaincatalog.Values) { value.Name = " " }},
		{name: "unknown purpose", mutate: func(value *domaincatalog.Values) { value.Purpose = "sale" }},
		{name: "unknown status", mutate: func(value *domaincatalog.Values) { value.Status = "deleted" }},
		{name: "empty tag", mutate: func(value *domaincatalog.Values) { value.Tags = []string{" "} }},
		{name: "too many tags", mutate: func(value *domaincatalog.Values) {
			value.Tags = make([]string, maximumTags+1)
			for index := range value.Tags {
				value.Tags[index] = "tag"
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := valid
			test.mutate(&input)
			if _, err := service.Create(context.Background(), input); !errors.Is(err, ErrInvalidItem) {
				t.Fatalf("Create() error = %v, want ErrInvalidItem", err)
			}
		})
	}
}

func TestListDefaultsPaginationAndValidatesFilters(t *testing.T) {
	repository := &catalogRepositoryStub{}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	purpose := domaincatalog.PurposePrototype
	status := domaincatalog.StatusActive
	if _, err := service.List(context.Background(), domaincatalog.ListFilter{
		Purpose: &purpose, Status: &status, Tag: " PLA ", Query: " cube ",
	}); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if repository.filter.Limit != DefaultListLimit || repository.filter.Tag != "pla" || repository.filter.Query != "cube" {
		t.Fatalf("normalized filter = %#v", repository.filter)
	}

	invalidPurpose := domaincatalog.Purpose("invalid")
	invalidStatus := domaincatalog.Status("invalid")
	for _, filter := range []domaincatalog.ListFilter{
		{Limit: MaximumListLimit + 1}, {Limit: 1, Offset: -1},
		{Limit: 1, Purpose: &invalidPurpose}, {Limit: 1, Status: &invalidStatus},
	} {
		if _, err := service.List(context.Background(), filter); !errors.Is(err, ErrInvalidListFilter) {
			t.Fatalf("List(%#v) error = %v, want ErrInvalidListFilter", filter, err)
		}
	}
}

func TestCatalogServicePreservesNotFoundErrors(t *testing.T) {
	repository := &catalogRepositoryStub{err: domaincatalog.ErrItemNotFound}
	service, err := NewService(repository)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	values := domaincatalog.Values{Name: "Part", Purpose: domaincatalog.PurposeProduct, Status: domaincatalog.StatusActive}
	id := "11111111-1111-4111-8111-111111111111"
	if _, err := service.Get(context.Background(), id); !errors.Is(err, domaincatalog.ErrItemNotFound) {
		t.Fatalf("Get() error = %v", err)
	}
	if _, err := service.Update(context.Background(), id, values); !errors.Is(err, domaincatalog.ErrItemNotFound) {
		t.Fatalf("Update() error = %v", err)
	}
	if err := service.Delete(context.Background(), id); !errors.Is(err, domaincatalog.ErrItemNotFound) {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := service.Get(context.Background(), "not-a-uuid"); !errors.Is(err, domaincatalog.ErrItemNotFound) {
		t.Fatalf("Get(invalid UUID) error = %v", err)
	}
}

type catalogRepositoryStub struct {
	created   domaincatalog.Values
	createdAt time.Time
	filter    domaincatalog.ListFilter
	err       error
}

func (stub *catalogRepositoryStub) Create(_ context.Context, values domaincatalog.Values, now time.Time) (domaincatalog.Item, error) {
	stub.created, stub.createdAt = values, now
	return domaincatalog.Item{Name: values.Name, Tags: values.Tags}, stub.err
}

func (stub *catalogRepositoryStub) FindByID(context.Context, string) (domaincatalog.Item, error) {
	return domaincatalog.Item{}, stub.err
}

func (stub *catalogRepositoryStub) List(_ context.Context, filter domaincatalog.ListFilter) (domaincatalog.Page, error) {
	stub.filter = filter
	return domaincatalog.Page{Items: []domaincatalog.Item{}, Limit: filter.Limit, Offset: filter.Offset}, stub.err
}

func (stub *catalogRepositoryStub) Update(context.Context, string, domaincatalog.Values, time.Time) (domaincatalog.Item, error) {
	return domaincatalog.Item{}, stub.err
}

func (stub *catalogRepositoryStub) Delete(context.Context, string) error {
	return stub.err
}
