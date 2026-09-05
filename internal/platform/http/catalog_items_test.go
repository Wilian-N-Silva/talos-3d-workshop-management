package httpplatform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	applicationauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/auth"
	applicationcatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/catalog"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
)

func TestCatalogCreateRequiresWritePermission(t *testing.T) {
	createdAt := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	service := &catalogItemServiceStub{item: domaincatalog.Item{
		ID: "item-id", Name: "Calibration Cube", Purpose: domaincatalog.PurposeTest,
		Tags: []string{"calibration"}, Status: domaincatalog.StatusActive,
		CreatedAt: createdAt, UpdatedAt: createdAt,
	}}
	body := `{"name":"Calibration Cube","description":"","purpose":"test","sellable":false,"tags":["calibration"]}`
	router := NewAPIV1Router()
	RegisterCatalogItems(router, authorizedCatalogUser(domainauth.RoleDesigner), service)
	request := catalogRequest(http.MethodPost, CatalogItemsPath, body)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || service.created.Name != "Calibration Cube" || service.created.Status != domaincatalog.StatusActive {
		t.Fatalf("create status/input = %d, %#v, body = %s", response.Code, service.created, response.Body.String())
	}
	var got catalogItemResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil || got.ID != "item-id" || got.Status != domaincatalog.StatusActive {
		t.Fatalf("create response = %#v, %v", got, err)
	}

	forbiddenRouter := NewAPIV1Router()
	RegisterCatalogItems(forbiddenRouter, authorizedCatalogUser(domainauth.RoleViewer), service)
	forbidden := httptest.NewRecorder()
	forbiddenRouter.ServeHTTP(forbidden, catalogRequest(http.MethodPost, CatalogItemsPath, body))
	assertAPIError(t, forbidden, http.StatusForbidden, "forbidden", "Permission denied")
}

func TestCatalogListParsesPaginationAndFilters(t *testing.T) {
	service := &catalogItemServiceStub{page: domaincatalog.Page{
		Items: []domaincatalog.Item{{ID: "one", Name: "Cube", Tags: nil, Purpose: domaincatalog.PurposeProduct, Status: domaincatalog.StatusActive}},
		Total: 7, Limit: 25, Offset: 50,
	}}
	router := NewAPIV1Router()
	RegisterCatalogItems(router, authorizedCatalogUser(domainauth.RoleViewer), service)
	request := catalogRequest(http.MethodGet, CatalogItemsPath+"?purpose=product&status=active&sellable=true&tag=pla&q=cube&limit=25&offset=50", "")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", response.Code, response.Body.String())
	}
	if service.filter.Purpose == nil || *service.filter.Purpose != domaincatalog.PurposeProduct ||
		service.filter.Status == nil || *service.filter.Status != domaincatalog.StatusActive ||
		service.filter.Sellable == nil || !*service.filter.Sellable ||
		service.filter.Tag != "pla" || service.filter.Query != "cube" || service.filter.Limit != 25 || service.filter.Offset != 50 {
		t.Fatalf("list filter = %#v", service.filter)
	}
	var got catalogListResponse
	if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil || got.Pagination.Total != 7 || len(got.Items) != 1 || got.Items[0].Tags == nil {
		t.Fatalf("list response = %#v, %v", got, err)
	}
}

func TestCatalogRoutesMapValidationAndNotFound(t *testing.T) {
	service := &catalogItemServiceStub{err: domaincatalog.ErrItemNotFound}
	router := NewAPIV1Router()
	RegisterCatalogItems(router, authorizedCatalogUser(domainauth.RoleDesigner), service)

	notFound := httptest.NewRecorder()
	router.ServeHTTP(notFound, catalogRequest(http.MethodGet, CatalogItemsPath+"/missing", ""))
	assertAPIError(t, notFound, http.StatusNotFound, "catalog_item_not_found", "Catalog item not found")

	service.err = nil
	invalidBody := httptest.NewRecorder()
	router.ServeHTTP(invalidBody, catalogRequest(http.MethodPost, CatalogItemsPath, `{"name":"part","unknown":true}`))
	assertAPIError(t, invalidBody, http.StatusBadRequest, "invalid_catalog_item", "Invalid catalog item")

	invalidFilter := httptest.NewRecorder()
	router.ServeHTTP(invalidFilter, catalogRequest(http.MethodGet, CatalogItemsPath+"?sellable=maybe", ""))
	assertAPIError(t, invalidFilter, http.StatusBadRequest, "invalid_catalog_filter", "Invalid catalog filter")

	service.err = applicationcatalog.ErrInvalidItem
	invalidValues := httptest.NewRecorder()
	router.ServeHTTP(invalidValues, catalogRequest(http.MethodPost, CatalogItemsPath, `{"name":"","purpose":"product"}`))
	assertAPIError(t, invalidValues, http.StatusBadRequest, "invalid_catalog_item", "Invalid catalog item")
}

func TestCatalogUpdateAndDeleteUseItemID(t *testing.T) {
	service := &catalogItemServiceStub{item: domaincatalog.Item{ID: "item-id", Name: "Updated", Tags: []string{}, Purpose: domaincatalog.PurposeInternal, Status: domaincatalog.StatusArchived}}
	router := NewAPIV1Router()
	RegisterCatalogItems(router, authorizedCatalogUser(domainauth.RoleDesigner), service)
	body := `{"name":"Updated","description":"","purpose":"internal","sellable":false,"tags":[],"status":"archived"}`
	updated := httptest.NewRecorder()
	router.ServeHTTP(updated, catalogRequest(http.MethodPut, CatalogItemsPath+"/item-id", body))
	if updated.Code != http.StatusOK || service.updatedID != "item-id" || service.updated.Status != domaincatalog.StatusArchived {
		t.Fatalf("update status/id/input = %d, %q, %#v", updated.Code, service.updatedID, service.updated)
	}

	deleted := httptest.NewRecorder()
	router.ServeHTTP(deleted, catalogRequest(http.MethodDelete, CatalogItemsPath+"/item-id", ""))
	if deleted.Code != http.StatusNoContent || service.deletedID != "item-id" {
		t.Fatalf("delete status/id = %d, %q", deleted.Code, service.deletedID)
	}
}

func catalogRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer token")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func authorizedCatalogUser(role domainauth.Role) *bearerAuthenticationServiceStub {
	return &bearerAuthenticationServiceStub{result: applicationauth.AuthenticationResult{
		User: domainauth.User{ID: "user-id", Status: domainauth.UserStatusActive, Role: role},
	}}
}

type catalogItemServiceStub struct {
	item      domaincatalog.Item
	page      domaincatalog.Page
	err       error
	created   domaincatalog.Values
	filter    domaincatalog.ListFilter
	updatedID string
	updated   domaincatalog.Values
	deletedID string
}

func (stub *catalogItemServiceStub) Create(_ context.Context, values domaincatalog.Values) (domaincatalog.Item, error) {
	stub.created = values
	return stub.item, stub.err
}

func (stub *catalogItemServiceStub) Get(context.Context, string) (domaincatalog.Item, error) {
	return stub.item, stub.err
}

func (stub *catalogItemServiceStub) List(_ context.Context, filter domaincatalog.ListFilter) (domaincatalog.Page, error) {
	stub.filter = filter
	return stub.page, stub.err
}

func (stub *catalogItemServiceStub) Update(_ context.Context, id string, values domaincatalog.Values) (domaincatalog.Item, error) {
	stub.updatedID, stub.updated = id, values
	return stub.item, stub.err
}

func (stub *catalogItemServiceStub) Delete(_ context.Context, id string) error {
	stub.deletedID = id
	return stub.err
}
