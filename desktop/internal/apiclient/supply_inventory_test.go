package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const supplyJSON = `{"id":"11111111-1111-4111-8111-111111111111","name":"NFC","sku":"NFC-01","unit":"unit","current_quantity":"20.000000","replacement_unit_cost_cents":75,"minimum_quantity":"10.000000","notes":"","created_at":"2026-09-04T15:00:00Z","updated_at":"2026-09-04T15:00:00Z"}`
const supplyMovementJSON = `{"id":"22222222-2222-4222-8222-222222222222","supply_id":"11111111-1111-4111-8111-111111111111","type":"purchase","quantity":"20.000000","unit_cost_cents":60,"reference_type":null,"reference_id":null,"occurred_at":"2026-09-04T15:00:00Z","recorded_by":"33333333-3333-4333-8333-333333333333","notes":"","created_at":"2026-09-04T15:00:00Z"}`

func TestClientSupplyWorkflowUsesBearerToken(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.Header.Get("Authorization") != "Bearer inventory-token" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		switch requestCount {
		case 1:
			if r.Method != http.MethodGet || r.URL.Path != "/api/v1/inventory/supplies" {
				t.Fatalf("list request=%s %s", r.Method, r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"supplies":[` + supplyJSON + `]}`))
		case 2:
			var input SupplyInput
			if r.Method != http.MethodPost || json.NewDecoder(r.Body).Decode(&input) != nil || input.Name != "NFC" {
				t.Fatalf("create request=%s input=%#v", r.Method, input)
			}
			_, _ = w.Write([]byte(supplyJSON))
		case 3:
			if r.URL.Path != "/api/v1/inventory/supplies/supply-id/movements" {
				t.Fatalf("movement path=%s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"movements":[` + supplyMovementJSON + `]}`))
		case 4:
			var input SupplyMovementInput
			if r.Method != http.MethodPost || r.URL.Path != "/api/v1/inventory/supplies/supply-id/movements" || json.NewDecoder(r.Body).Decode(&input) != nil || input.Quantity != "20" {
				t.Fatalf("record movement request=%s %s input=%#v", r.Method, r.URL.Path, input)
			}
			_, _ = w.Write([]byte(supplyMovementJSON))
		case 5:
			if r.URL.Path != "/api/v1/inventory/low-stock" || r.URL.Query().Get("spool_threshold_g") != "75" {
				t.Fatalf("low inventory target=%s", r.URL.String())
			}
			_, _ = w.Write([]byte(`{"spool_threshold_g":"75","spools":[],"supplies":[` + supplyJSON + `]}`))
		}
	}))
	defer server.Close()
	client, _ := New(server.URL, "1.0.0")
	ctx := context.Background()
	if values, err := client.ListSupplies(ctx, "inventory-token"); err != nil || len(values) != 1 {
		t.Fatalf("ListSupplies()=%#v,%v", values, err)
	}
	if value, err := client.CreateSupply(ctx, "inventory-token", SupplyInput{Name: "NFC"}); err != nil || value.Name != "NFC" {
		t.Fatalf("CreateSupply()=%#v,%v", value, err)
	}
	if values, err := client.ListSupplyMovements(ctx, "inventory-token", "supply-id"); err != nil || len(values) != 1 {
		t.Fatalf("ListSupplyMovements()=%#v,%v", values, err)
	}
	if value, err := client.RecordSupplyMovement(ctx, "inventory-token", "supply-id", SupplyMovementInput{Quantity: "20"}); err != nil || value.Quantity != "20.000000" {
		t.Fatalf("RecordSupplyMovement()=%#v,%v", value, err)
	}
	if value, err := client.ListLowInventory(ctx, "inventory-token", "75"); err != nil || value.SpoolThresholdG != "75" || len(value.Supplies) != 1 {
		t.Fatalf("ListLowInventory()=%#v,%v", value, err)
	}
}
