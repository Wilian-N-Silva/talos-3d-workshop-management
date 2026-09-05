package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const catalogPartJSON = `{"id":"22222222-2222-4222-8222-222222222222","catalog_item_id":"11111111-1111-4111-8111-111111111111","name":"Body","quantity":2,"notes":"","created_at":"2026-09-04T12:00:00Z","updated_at":"2026-09-04T12:00:00Z"}`
const designVersionJSON = `{"id":"33333333-3333-4333-8333-333333333333","catalog_part_id":"22222222-2222-4222-8222-222222222222","version":"v1","notes":"","origin":"third_party","source_url":"https://example.com/model","original_author":"Maker","license_name":"CC BY-NC","commercial_use_allowed":false,"attribution_required":true,"attribution_text":"Maker","created_by":"55555555-5555-4555-8555-555555555555","created_at":"2026-09-04T12:00:00Z","files":[]}`
const designFileJSON = `{"file_id":"44444444-4444-4444-8444-444444444444","role":"print","original_name":"body.3mf","content_type":"application/octet-stream","size_bytes":123,"sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","created_at":"2026-09-04T12:00:00Z"}`

func TestClientCatalogDesignWorkflowUsesNativeBearerRequests(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		if request.Header.Get("Authorization") != "Bearer secure-token" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		switch requests {
		case 1:
			if request.Method != http.MethodGet || request.URL.Path != "/api/v1/catalog/items/item-id/parts" {
				t.Fatalf("parts request = %s %s", request.Method, request.URL.Path)
			}
			_, _ = response.Write([]byte(`{"parts":[` + catalogPartJSON + `]}`))
		case 2:
			if request.Method != http.MethodPost || request.URL.Path != "/api/v1/catalog/items/item-id/parts" {
				t.Fatalf("create part request = %s %s", request.Method, request.URL.Path)
			}
			_, _ = response.Write([]byte(catalogPartJSON))
		case 3:
			if request.Method != http.MethodGet || request.URL.Path != "/api/v1/catalog/parts/part-id/design-versions" {
				t.Fatalf("history request = %s %s", request.Method, request.URL.Path)
			}
			_, _ = response.Write([]byte(`{"versions":[` + designVersionJSON + `]}`))
		case 4:
			var input DesignVersionInput
			if err := json.NewDecoder(request.Body).Decode(&input); err != nil || input.Version != "v2" || input.CommercialUseAllowed == nil || !*input.CommercialUseAllowed {
				t.Fatalf("version input = %#v, %v", input, err)
			}
			_, _ = response.Write([]byte(designVersionJSON))
		case 5:
			var input map[string]string
			if err := json.NewDecoder(request.Body).Decode(&input); err != nil || input["file_id"] != "file-id" || input["role"] != "print" {
				t.Fatalf("file input = %#v, %v", input, err)
			}
			_, _ = response.Write([]byte(designFileJSON))
		}
	}))
	defer server.Close()
	client, _ := New(server.URL, "1.0.0")
	ctx := context.Background()
	if parts, err := client.ListCatalogParts(ctx, "secure-token", "item-id"); err != nil || len(parts) != 1 {
		t.Fatalf("ListCatalogParts() = %#v, %v", parts, err)
	}
	if _, err := client.CreateCatalogPart(ctx, "secure-token", "item-id", CatalogPartInput{Name: "Body", Quantity: 2}); err != nil {
		t.Fatalf("CreateCatalogPart() error = %v", err)
	}
	if versions, err := client.ListDesignVersions(ctx, "secure-token", "part-id"); err != nil || len(versions) != 1 || versions[0].CommercialUseAllowed == nil || *versions[0].CommercialUseAllowed {
		t.Fatalf("ListDesignVersions() = %#v, %v", versions, err)
	}
	allowed := true
	if _, err := client.CreateDesignVersion(ctx, "secure-token", "part-id", DesignVersionInput{Version: "v2", Origin: "original", CommercialUseAllowed: &allowed}); err != nil {
		t.Fatalf("CreateDesignVersion() error = %v", err)
	}
	if file, err := client.AttachDesignFile(ctx, "secure-token", "version-id", "file-id", "print"); err != nil || file.Role != "print" {
		t.Fatalf("AttachDesignFile() = %#v, %v", file, err)
	}
}

func TestClientRejectsInvalidDesignFileDigest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write([]byte(strings.Replace(designFileJSON, strings.Repeat("a", 64), strings.Repeat("z", 64), 1)))
	}))
	defer server.Close()
	client, _ := New(server.URL, "1.0.0")
	_, err := client.AttachDesignFile(context.Background(), "secure-token", "version-id", "file-id", "print")
	assertClientErrorKind(t, err, ErrorInvalidResponse)
}
