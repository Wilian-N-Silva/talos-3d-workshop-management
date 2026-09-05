package httpplatform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	applicationcatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/catalog"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"
)

func TestCatalogPartsCRUDAndPermissions(t *testing.T) {
	service := &catalogDesignServiceStub{part: domaincatalog.Part{ID: "part-id", CatalogItemID: "item-id", Name: "Body", Quantity: 2}}
	router := NewAPIV1Router()
	RegisterCatalogDesigns(router, authorizedCatalogUser(domainauth.RoleDesigner), service)

	created := serveCatalogDesign(t, router, http.MethodPost, CatalogItemsPath+"/item-id/parts", `{"name":"Body","quantity":2,"notes":"two pieces"}`)
	if created.Code != http.StatusCreated || service.itemID != "item-id" || service.partValues.Quantity != 2 {
		t.Fatalf("create part = %d, item = %q, values = %#v, body = %s", created.Code, service.itemID, service.partValues, created.Body.String())
	}
	listed := serveCatalogDesign(t, router, http.MethodGet, CatalogItemsPath+"/item-id/parts", "")
	if listed.Code != http.StatusOK {
		t.Fatalf("list parts = %d: %s", listed.Code, listed.Body.String())
	}
	updated := serveCatalogDesign(t, router, http.MethodPut, CatalogPartsPath+"/part-id", `{"name":"Shell","quantity":1,"notes":""}`)
	if updated.Code != http.StatusOK || service.partID != "part-id" || service.partValues.Name != "Shell" {
		t.Fatalf("update part = %d, %#v", updated.Code, service.partValues)
	}
	deleted := serveCatalogDesign(t, router, http.MethodDelete, CatalogPartsPath+"/part-id", "")
	if deleted.Code != http.StatusNoContent || service.deletedPartID != "part-id" {
		t.Fatalf("delete part = %d, %q", deleted.Code, service.deletedPartID)
	}

	forbiddenRouter := NewAPIV1Router()
	RegisterCatalogDesigns(forbiddenRouter, authorizedCatalogUser(domainauth.RoleViewer), service)
	forbidden := serveCatalogDesign(t, forbiddenRouter, http.MethodPost, CatalogItemsPath+"/item-id/parts", `{"name":"Body","quantity":1}`)
	assertAPIError(t, forbidden, http.StatusForbidden, "forbidden", "Permission denied")
}

func TestDesignVersionCreationHistoryAndFileRoles(t *testing.T) {
	denied := false
	createdAt := time.Date(2026, 9, 4, 18, 0, 0, 0, time.UTC)
	service := &catalogDesignServiceStub{version: domaincatalog.DesignVersion{
		ID: "version-id", CatalogPartID: "part-id", Version: "v1", Origin: domaincatalog.DesignOriginThirdParty,
		CommercialUseAllowed: &denied, CreatedBy: "user-id", CreatedAt: createdAt, Files: []domaincatalog.DesignFile{},
	}, file: domaincatalog.DesignFile{FileID: "file-id", Role: domaincatalog.DesignFilePrint, OriginalName: "part.3mf", SHA256Hex: strings.Repeat("a", 64), CreatedAt: createdAt}}
	router := NewAPIV1Router()
	RegisterCatalogDesigns(router, authorizedCatalogUser(domainauth.RoleDesigner), service)
	body := `{"version":"v1","notes":"first","origin":"third_party","source_url":"https://example.com/model","original_author":"Maker","license_name":"NC","commercial_use_allowed":false,"attribution_required":true,"attribution_text":"Maker"}`
	created := serveCatalogDesign(t, router, http.MethodPost, CatalogPartsPath+"/part-id/design-versions", body)
	if created.Code != http.StatusCreated || service.partID != "part-id" || service.actorID != "user-id" || service.versionValues.CommercialUseAllowed == nil || *service.versionValues.CommercialUseAllowed {
		t.Fatalf("create version = %d, part/actor = %q/%q, values = %#v, body = %s", created.Code, service.partID, service.actorID, service.versionValues, created.Body.String())
	}
	var response designVersionResponse
	if err := json.Unmarshal(created.Body.Bytes(), &response); err != nil || response.CommercialUseAllowed == nil || *response.CommercialUseAllowed || response.Files == nil {
		t.Fatalf("version response = %#v, %v", response, err)
	}
	history := serveCatalogDesign(t, router, http.MethodGet, CatalogPartsPath+"/part-id/design-versions", "")
	if history.Code != http.StatusOK {
		t.Fatalf("version history = %d: %s", history.Code, history.Body.String())
	}
	attached := serveCatalogDesign(t, router, http.MethodPost, DesignVersionsPath+"/version-id/files", `{"file_id":"file-id","role":"print"}`)
	if attached.Code != http.StatusCreated || service.versionID != "version-id" || service.fileID != "file-id" || service.role != domaincatalog.DesignFilePrint {
		t.Fatalf("attach file = %d, version/file/role = %q/%q/%q", attached.Code, service.versionID, service.fileID, service.role)
	}
}

func TestCatalogDesignErrorsAreStable(t *testing.T) {
	service := &catalogDesignServiceStub{err: domaincatalog.ErrDesignVersionConflict}
	router := NewAPIV1Router()
	RegisterCatalogDesigns(router, authorizedCatalogUser(domainauth.RoleDesigner), service)
	response := serveCatalogDesign(t, router, http.MethodPost, CatalogPartsPath+"/part-id/design-versions", `{"version":"v1","origin":"unknown"}`)
	assertAPIError(t, response, http.StatusConflict, "design_version_exists", "Design version already exists")
	service.err = applicationcatalog.ErrInvalidDesignFile
	response = serveCatalogDesign(t, router, http.MethodPost, DesignVersionsPath+"/version-id/files", `{"file_id":"file-id","role":"bad"}`)
	assertAPIError(t, response, http.StatusBadRequest, "invalid_catalog_design", "Invalid catalog design data")
}

func TestDesignVersionsHaveNoOverwriteRoute(t *testing.T) {
	service := &catalogDesignServiceStub{}
	router := NewAPIV1Router()
	RegisterCatalogDesigns(router, authorizedCatalogUser(domainauth.RoleDesigner), service)
	response := serveCatalogDesign(t, router, http.MethodPut, DesignVersionsPath+"/version-id", `{"version":"v2","origin":"original"}`)
	assertAPIError(t, response, http.StatusNotFound, "route_not_found", "Route not found")
}

func TestCatalogPartDeletePreservesImmutableDesignHistory(t *testing.T) {
	service := &catalogDesignServiceStub{err: domaincatalog.ErrDesignHistoryExists}
	router := NewAPIV1Router()
	RegisterCatalogDesigns(router, authorizedCatalogUser(domainauth.RoleDesigner), service)
	response := serveCatalogDesign(t, router, http.MethodDelete, CatalogPartsPath+"/part-id", "")
	assertAPIError(t, response, http.StatusConflict, "design_history_exists", "Immutable design history prevents deletion")
}

func serveCatalogDesign(t *testing.T, router http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer token")
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

type catalogDesignServiceStub struct {
	part          domaincatalog.Part
	version       domaincatalog.DesignVersion
	file          domaincatalog.DesignFile
	err           error
	itemID        string
	partID        string
	versionID     string
	fileID        string
	actorID       string
	deletedPartID string
	partValues    domaincatalog.PartValues
	versionValues domaincatalog.DesignVersionValues
	role          domaincatalog.DesignFileRole
}

func (stub *catalogDesignServiceStub) CreatePart(_ context.Context, itemID string, values domaincatalog.PartValues) (domaincatalog.Part, error) {
	stub.itemID, stub.partValues = itemID, values
	return stub.part, stub.err
}
func (stub *catalogDesignServiceStub) GetPart(context.Context, string) (domaincatalog.Part, error) {
	return stub.part, stub.err
}
func (stub *catalogDesignServiceStub) ListParts(_ context.Context, itemID string) ([]domaincatalog.Part, error) {
	stub.itemID = itemID
	return []domaincatalog.Part{stub.part}, stub.err
}
func (stub *catalogDesignServiceStub) UpdatePart(_ context.Context, partID string, values domaincatalog.PartValues) (domaincatalog.Part, error) {
	stub.partID, stub.partValues = partID, values
	stub.part.Name = values.Name
	return stub.part, stub.err
}
func (stub *catalogDesignServiceStub) DeletePart(_ context.Context, partID string) error {
	stub.deletedPartID = partID
	return stub.err
}
func (stub *catalogDesignServiceStub) CreateVersion(_ context.Context, partID, actorID string, values domaincatalog.DesignVersionValues) (domaincatalog.DesignVersion, error) {
	stub.partID, stub.actorID, stub.versionValues = partID, actorID, values
	return stub.version, stub.err
}
func (stub *catalogDesignServiceStub) ListVersions(_ context.Context, partID string) ([]domaincatalog.DesignVersion, error) {
	stub.partID = partID
	return []domaincatalog.DesignVersion{stub.version}, stub.err
}
func (stub *catalogDesignServiceStub) AttachFile(_ context.Context, versionID, fileID, actorID string, role domaincatalog.DesignFileRole) (domaincatalog.DesignFile, error) {
	stub.versionID, stub.fileID, stub.actorID, stub.role = versionID, fileID, actorID, role
	return stub.file, stub.err
}
