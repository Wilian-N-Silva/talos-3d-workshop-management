package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const catalogItemsPath = "/api/v1/catalog/items"

// CatalogItem is safe catalog metadata exposed to the desktop WebView.
type CatalogItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	SKU         *string  `json:"sku"`
	Description string   `json:"description"`
	Purpose     string   `json:"purpose"`
	Sellable    bool     `json:"sellable"`
	Tags        []string `json:"tags"`
	Status      string   `json:"status"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// CatalogItemInput contains editable catalog fields.
type CatalogItemInput struct {
	Name        string   `json:"name"`
	SKU         *string  `json:"sku"`
	Description string   `json:"description"`
	Purpose     string   `json:"purpose"`
	Sellable    bool     `json:"sellable"`
	Tags        []string `json:"tags"`
	Status      string   `json:"status"`
}

// CatalogPage contains one bounded catalog list page.
type CatalogPage struct {
	Items      []CatalogItem     `json:"items"`
	Pagination CatalogPagination `json:"pagination"`
}

// CatalogPagination describes one bounded list window.
type CatalogPagination struct {
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Total  int64 `json:"total"`
}

// ListCatalogItems returns the first bounded desktop catalog page.
func (client *Client) ListCatalogItems(ctx context.Context, token string) (CatalogPage, error) {
	var page CatalogPage
	err := client.catalogJSON(ctx, http.MethodGet, catalogItemsPath+"?limit=100&offset=0", token, nil, &page)
	if err != nil {
		return CatalogPage{}, err
	}
	if page.Items == nil || page.Pagination.Limit < 1 || page.Pagination.Limit > 100 || page.Pagination.Offset < 0 || page.Pagination.Total < 0 {
		return CatalogPage{}, invalidResponseError("Server returned invalid catalog data", nil)
	}
	for _, item := range page.Items {
		if !validCatalogItem(item) {
			return CatalogPage{}, invalidResponseError("Server returned invalid catalog data", nil)
		}
	}
	return page, nil
}

// CreateCatalogItem creates one item through the authenticated API.
func (client *Client) CreateCatalogItem(ctx context.Context, token string, input CatalogItemInput) (CatalogItem, error) {
	var item CatalogItem
	if err := client.catalogJSON(ctx, http.MethodPost, catalogItemsPath, token, input, &item); err != nil {
		return CatalogItem{}, err
	}
	if !validCatalogItem(item) {
		return CatalogItem{}, invalidResponseError("Server returned invalid catalog data", nil)
	}
	return item, nil
}

// UpdateCatalogItem replaces editable fields for one item.
func (client *Client) UpdateCatalogItem(ctx context.Context, token, id string, input CatalogItemInput) (CatalogItem, error) {
	var item CatalogItem
	path := catalogItemsPath + "/" + url.PathEscape(strings.TrimSpace(id))
	if err := client.catalogJSON(ctx, http.MethodPut, path, token, input, &item); err != nil {
		return CatalogItem{}, err
	}
	if !validCatalogItem(item) {
		return CatalogItem{}, invalidResponseError("Server returned invalid catalog data", nil)
	}
	return item, nil
}

func (client *Client) catalogJSON(ctx context.Context, method, path, token string, input, output any) error {
	if strings.TrimSpace(token) == "" {
		return errors.New("missing session token")
	}
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return invalidResponseError("Unable to encode catalog request", err)
		}
		body = bytes.NewReader(encoded)
	}
	requestContext, cancel := context.WithTimeout(ctx, client.timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, method, client.baseURL+path, body)
	if err != nil {
		return invalidResponseError("Unable to create catalog request", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return mapTransportError(requestContext, err, "Catalog request timed out")
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maximumResponseSize+1))
	if err != nil {
		return invalidResponseError("Unable to read catalog response", err)
	}
	if len(responseBody) > maximumResponseSize {
		return invalidResponseError("Server response is too large", nil)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return mapAPIError(response.StatusCode, responseBody)
	}
	if err := json.Unmarshal(responseBody, output); err != nil {
		return invalidResponseError("Server returned invalid catalog data", err)
	}
	return nil
}

func validCatalogItem(item CatalogItem) bool {
	if strings.TrimSpace(item.ID) == "" || strings.TrimSpace(item.Name) == "" {
		return false
	}
	if _, err := time.Parse(time.RFC3339, item.CreatedAt); err != nil {
		return false
	}
	if _, err := time.Parse(time.RFC3339, item.UpdatedAt); err != nil {
		return false
	}
	switch item.Purpose {
	case "product", "prototype", "tooling", "test", "internal", "personal":
	default:
		return false
	}
	return item.Status == "active" || item.Status == "archived"
}
