package httpplatform

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	applicationcatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/application/catalog"
	domainauth "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/auth"
	domaincatalog "github.com/Wilian-N-Silva/talos-3d-workshop-management/internal/domain/catalog"

	"github.com/go-chi/chi/v5"
)

const (
	CatalogItemsPath = "/catalog/items"
	catalogItemPath  = CatalogItemsPath + "/{itemID}"
	catalogBodyLimit = 64 * 1024
)

// CatalogItemService manages validated catalog item CRUD and list operations.
type CatalogItemService interface {
	Create(context.Context, domaincatalog.Values) (domaincatalog.Item, error)
	Get(context.Context, string) (domaincatalog.Item, error)
	List(context.Context, domaincatalog.ListFilter) (domaincatalog.Page, error)
	Update(context.Context, string, domaincatalog.Values) (domaincatalog.Item, error)
	Delete(context.Context, string) error
}

type catalogItemRequest struct {
	Name        string                `json:"name"`
	SKU         *string               `json:"sku"`
	Description string                `json:"description"`
	Purpose     domaincatalog.Purpose `json:"purpose"`
	Sellable    bool                  `json:"sellable"`
	Tags        []string              `json:"tags"`
	Status      domaincatalog.Status  `json:"status"`
}

type catalogItemResponse struct {
	ID          string                `json:"id"`
	Name        string                `json:"name"`
	SKU         *string               `json:"sku"`
	Description string                `json:"description"`
	Purpose     domaincatalog.Purpose `json:"purpose"`
	Sellable    bool                  `json:"sellable"`
	Tags        []string              `json:"tags"`
	Status      domaincatalog.Status  `json:"status"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

type catalogPaginationResponse struct {
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Total  int64 `json:"total"`
}

type catalogListResponse struct {
	Items      []catalogItemResponse     `json:"items"`
	Pagination catalogPaginationResponse `json:"pagination"`
}

// RegisterCatalogItems registers permission-protected catalog item CRUD routes.
func RegisterCatalogItems(
	router *APIV1Router,
	authentication BearerAuthenticationService,
	service CatalogItemService,
) {
	router.Handle(http.MethodGet, CatalogItemsPath, RequirePermission(
		authentication, domainauth.PermissionCatalogRead,
	)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		filter, err := catalogListFilter(request)
		if err != nil {
			writeInvalidCatalogFilter(response)
			return
		}
		page, err := service.List(request.Context(), filter)
		if errors.Is(err, applicationcatalog.ErrInvalidListFilter) {
			writeInvalidCatalogFilter(response)
			return
		}
		if err != nil {
			WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
			return
		}
		items := make([]catalogItemResponse, 0, len(page.Items))
		for _, item := range page.Items {
			items = append(items, newCatalogItemResponse(item))
		}
		writeJSON(response, http.StatusOK, catalogListResponse{
			Items: items,
			Pagination: catalogPaginationResponse{
				Limit: page.Limit, Offset: page.Offset, Total: page.Total,
			},
		})
	})))

	router.Handle(http.MethodPost, CatalogItemsPath, RequirePermission(
		authentication, domainauth.PermissionCatalogWrite,
	)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		body, ok := decodeCatalogItemRequest(response, request)
		if !ok {
			return
		}
		item, err := service.Create(request.Context(), body.values())
		if !writeCatalogMutationResult(response, item, err, http.StatusCreated) {
			return
		}
	})))

	router.Handle(http.MethodGet, catalogItemPath, RequirePermission(
		authentication, domainauth.PermissionCatalogRead,
	)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		item, err := service.Get(request.Context(), chi.URLParam(request, "itemID"))
		if !writeCatalogMutationResult(response, item, err, http.StatusOK) {
			return
		}
	})))

	router.Handle(http.MethodPut, catalogItemPath, RequirePermission(
		authentication, domainauth.PermissionCatalogWrite,
	)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		body, ok := decodeCatalogItemRequest(response, request)
		if !ok {
			return
		}
		item, err := service.Update(request.Context(), chi.URLParam(request, "itemID"), body.values())
		if !writeCatalogMutationResult(response, item, err, http.StatusOK) {
			return
		}
	})))

	router.Handle(http.MethodDelete, catalogItemPath, RequirePermission(
		authentication, domainauth.PermissionCatalogWrite,
	)(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		err := service.Delete(request.Context(), chi.URLParam(request, "itemID"))
		switch {
		case errors.Is(err, domaincatalog.ErrItemNotFound):
			writeCatalogNotFound(response)
		case errors.Is(err, domaincatalog.ErrDesignHistoryExists):
			WriteError(response, http.StatusConflict, "design_history_exists", "Immutable design history prevents deletion", nil)
		case err != nil:
			WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		default:
			response.WriteHeader(http.StatusNoContent)
		}
	})))
}

func (request catalogItemRequest) values() domaincatalog.Values {
	status := request.Status
	if status == "" {
		status = domaincatalog.StatusActive
	}
	return domaincatalog.Values{
		Name: request.Name, SKU: request.SKU, Description: request.Description,
		Purpose: request.Purpose, Sellable: request.Sellable, Tags: request.Tags, Status: status,
	}
}

func decodeCatalogItemRequest(response http.ResponseWriter, request *http.Request) (catalogItemRequest, bool) {
	request.Body = http.MaxBytesReader(response, request.Body, catalogBodyLimit)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var body catalogItemRequest
	if err := decoder.Decode(&body); err != nil {
		writeInvalidCatalogItem(response)
		return catalogItemRequest{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeInvalidCatalogItem(response)
		return catalogItemRequest{}, false
	}
	return body, true
}

func catalogListFilter(request *http.Request) (domaincatalog.ListFilter, error) {
	query := request.URL.Query()
	filter := domaincatalog.ListFilter{Tag: query.Get("tag"), Query: query.Get("q")}
	if raw := query.Get("purpose"); raw != "" {
		purpose := domaincatalog.Purpose(raw)
		filter.Purpose = &purpose
	}
	if raw := query.Get("status"); raw != "" {
		status := domaincatalog.Status(raw)
		filter.Status = &status
	}
	if raw := query.Get("sellable"); raw != "" {
		sellable, err := strconv.ParseBool(raw)
		if err != nil {
			return domaincatalog.ListFilter{}, err
		}
		filter.Sellable = &sellable
	}
	var err error
	if raw := query.Get("limit"); raw != "" {
		filter.Limit, err = strconv.Atoi(raw)
		if err != nil {
			return domaincatalog.ListFilter{}, err
		}
	}
	if raw := query.Get("offset"); raw != "" {
		filter.Offset, err = strconv.Atoi(raw)
		if err != nil {
			return domaincatalog.ListFilter{}, err
		}
	}
	return filter, nil
}

func writeCatalogMutationResult(response http.ResponseWriter, item domaincatalog.Item, err error, status int) bool {
	switch {
	case errors.Is(err, applicationcatalog.ErrInvalidItem):
		writeInvalidCatalogItem(response)
		return false
	case errors.Is(err, domaincatalog.ErrItemNotFound):
		writeCatalogNotFound(response)
		return false
	case err != nil:
		WriteError(response, http.StatusInternalServerError, "internal_error", "Internal server error", nil)
		return false
	default:
		writeJSON(response, status, newCatalogItemResponse(item))
		return true
	}
}

func newCatalogItemResponse(item domaincatalog.Item) catalogItemResponse {
	tags := item.Tags
	if tags == nil {
		tags = []string{}
	}
	return catalogItemResponse{
		ID: item.ID, Name: item.Name, SKU: item.SKU, Description: item.Description,
		Purpose: item.Purpose, Sellable: item.Sellable, Tags: tags, Status: item.Status,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}

func writeInvalidCatalogItem(response http.ResponseWriter) {
	WriteError(response, http.StatusBadRequest, "invalid_catalog_item", "Invalid catalog item", nil)
}

func writeInvalidCatalogFilter(response http.ResponseWriter) {
	WriteError(response, http.StatusBadRequest, "invalid_catalog_filter", "Invalid catalog filter", nil)
}

func writeCatalogNotFound(response http.ResponseWriter) {
	WriteError(response, http.StatusNotFound, "catalog_item_not_found", "Catalog item not found", nil)
}
