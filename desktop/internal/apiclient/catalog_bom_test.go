package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const catalogBOMItemJSON = `{"id":"11111111-1111-4111-8111-111111111111","catalog_item_id":"22222222-2222-4222-8222-222222222222","supply_id":"33333333-3333-4333-8333-333333333333","quantity_per_unit":"1.000000","waste_percent":"10.0000","notes":"","created_at":"2026-09-04T18:00:00Z","updated_at":"2026-09-04T18:00:00Z"}`

func TestClientCatalogBOMWorkflowUsesBearerToken(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.Header.Get("Authorization") != "Bearer catalog-token" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		expectedPath := "/api/v1/catalog/items/item-id/bom"
		if requestCount >= 3 {
			expectedPath += "/bom-id"
		}
		if r.URL.Path != expectedPath {
			t.Fatalf("request %d path=%s", requestCount, r.URL.Path)
		}
		switch requestCount {
		case 1:
			_, _ = w.Write([]byte(`{"items":[{` + catalogBOMItemJSON[1:len(catalogBOMItemJSON)-1] + `,"supply_name":"NFC","supply_unit":"unit","replacement_unit_cost_cents":75,"effective_quantity_per_unit":"1.1","exact_replacement_cost_cents_per_unit":"82.5"}],"exact_total_replacement_cost_cents":"82.5","rounding_applied":false}`))
		case 2, 3:
			var input CatalogBOMInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.QuantityPerUnit != "1" {
				t.Fatalf("input=%#v,%v", input, err)
			}
			_, _ = w.Write([]byte(catalogBOMItemJSON))
		case 4:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()
	client, _ := New(server.URL, "1.0.0")
	ctx := context.Background()
	if value, err := client.GetCatalogBOM(ctx, "catalog-token", "item-id"); err != nil || len(value.Items) != 1 || value.ExactTotalReplacementCostCents != "82.5" {
		t.Fatalf("GetCatalogBOM()=%#v,%v", value, err)
	}
	input := CatalogBOMInput{SupplyID: "supply-id", QuantityPerUnit: "1", WastePercent: "10"}
	if _, err := client.CreateCatalogBOMItem(ctx, "catalog-token", "item-id", input); err != nil {
		t.Fatalf("CreateCatalogBOMItem()=%v", err)
	}
	if _, err := client.UpdateCatalogBOMItem(ctx, "catalog-token", "item-id", "bom-id", input); err != nil {
		t.Fatalf("UpdateCatalogBOMItem()=%v", err)
	}
	if err := client.DeleteCatalogBOMItem(ctx, "catalog-token", "item-id", "bom-id"); err != nil {
		t.Fatalf("DeleteCatalogBOMItem()=%v", err)
	}
}
