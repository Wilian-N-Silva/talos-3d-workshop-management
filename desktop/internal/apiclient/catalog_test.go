package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const catalogItemJSON = `{"id":"11111111-1111-4111-8111-111111111111","name":"Cube","sku":null,"description":"Test","purpose":"test","sellable":false,"tags":["pla"],"status":"active","created_at":"2026-09-04T12:00:00Z","updated_at":"2026-09-04T12:00:00Z"}`

func TestClientListsCatalogItemsWithBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != catalogItemsPath || request.URL.Query().Get("limit") != "100" || request.Header.Get("Authorization") != "Bearer secure-token" {
			t.Fatalf("request = %s %s auth=%q", request.Method, request.URL.String(), request.Header.Get("Authorization"))
		}
		_, _ = response.Write([]byte(`{"items":[` + catalogItemJSON + `],"pagination":{"limit":100,"offset":0,"total":1}}`))
	}))
	defer server.Close()
	client, _ := New(server.URL, "1.0.0")
	page, err := client.ListCatalogItems(context.Background(), "secure-token")
	if err != nil || len(page.Items) != 1 || page.Items[0].Name != "Cube" || page.Pagination.Total != 1 {
		t.Fatalf("ListCatalogItems() = %#v, %v", page, err)
	}
}

func TestClientCreatesAndUpdatesCatalogItems(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		if request.Header.Get("Authorization") != "Bearer secure-token" || request.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("headers = %#v", request.Header)
		}
		var input CatalogItemInput
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil || input.Name != "Cube" {
			t.Fatalf("input = %#v, %v", input, err)
		}
		switch requests {
		case 1:
			if request.Method != http.MethodPost || request.URL.Path != catalogItemsPath {
				t.Fatalf("create request = %s %s", request.Method, request.URL.Path)
			}
		case 2:
			if request.Method != http.MethodPut || request.URL.Path != catalogItemsPath+"/11111111-1111-4111-8111-111111111111" {
				t.Fatalf("update request = %s %s", request.Method, request.URL.Path)
			}
		}
		response.WriteHeader(map[bool]int{true: http.StatusCreated, false: http.StatusOK}[requests == 1])
		_, _ = response.Write([]byte(catalogItemJSON))
	}))
	defer server.Close()
	client, _ := New(server.URL, "1.0.0")
	input := CatalogItemInput{Name: "Cube", Purpose: "test", Status: "active", Tags: []string{"pla"}}
	if _, err := client.CreateCatalogItem(context.Background(), "secure-token", input); err != nil {
		t.Fatalf("CreateCatalogItem() error = %v", err)
	}
	if _, err := client.UpdateCatalogItem(context.Background(), "secure-token", "11111111-1111-4111-8111-111111111111", input); err != nil {
		t.Fatalf("UpdateCatalogItem() error = %v", err)
	}
}

func TestClientRejectsInvalidCatalogResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(`{"items":[{"id":"bad"}],"pagination":{"limit":100,"offset":0,"total":1}}`))
	}))
	defer server.Close()
	client, _ := New(server.URL, "1.0.0")
	_, err := client.ListCatalogItems(context.Background(), "secure-token")
	assertClientErrorKind(t, err, ErrorInvalidResponse)
}
