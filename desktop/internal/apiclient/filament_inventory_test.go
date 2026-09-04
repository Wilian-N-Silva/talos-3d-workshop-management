package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const spoolJSON = `{"id":"11111111-1111-4111-8111-111111111111","code":"PLA-001","material_id":"22222222-2222-4222-8222-222222222222","nominal_net_weight_g":"1000.000","tare_weight_g":"250.000","gross_weight_at_open_g":null,"current_remaining_weight_g":"595.500","purchase_cost_cents":9990,"replacement_cost_per_kg_cents":12990,"opened_at":null,"last_weighed_at":"2026-09-04T12:00:00Z","last_dried_at":null,"storage_location":"Shelf","storage_status":"","lot_number":"","status":"open","created_at":"2026-09-04T10:00:00Z","updated_at":"2026-09-04T12:00:00Z"}`
const measurementJSON = `{"id":"33333333-3333-4333-8333-333333333333","spool_id":"11111111-1111-4111-8111-111111111111","measured_at":"2026-09-04T12:00:00Z","gross_weight_g":"845.500","derived_remaining_weight_g":"595.500","source":"manual","notes":"","recorded_by":"44444444-4444-4444-8444-444444444444","created_at":"2026-09-04T12:00:00Z"}`

func TestClientSpoolMeasurementWorkflowUsesBearerToken(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer inventory-token" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		switch requests {
		case 1:
			if r.Method != http.MethodGet || r.URL.Path != "/api/v1/inventory/spools" {
				t.Fatalf("spools request=%s %s", r.Method, r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"spools":[` + spoolJSON + `]}`))
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != "/api/v1/inventory/spools/spool-id/measurements" {
				t.Fatalf("history request=%s %s", r.Method, r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"measurements":[` + measurementJSON + `]}`))
		case 3:
			var input MeasurementInput
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.GrossWeightG != "845.5" {
				t.Fatalf("input=%#v,%v", input, err)
			}
			_, _ = w.Write([]byte(measurementJSON))
		}
	}))
	defer server.Close()
	client, _ := New(server.URL, "1.0.0")
	ctx := context.Background()
	if values, err := client.ListSpools(ctx, "inventory-token"); err != nil || len(values) != 1 || values[0].CurrentRemainingWeightG == nil {
		t.Fatalf("ListSpools()=%#v,%v", values, err)
	}
	if values, err := client.ListSpoolMeasurements(ctx, "inventory-token", "spool-id"); err != nil || len(values) != 1 {
		t.Fatalf("ListSpoolMeasurements()=%#v,%v", values, err)
	}
	if value, err := client.RecordSpoolMeasurement(ctx, "inventory-token", "spool-id", MeasurementInput{GrossWeightG: "845.5"}); err != nil || value.DerivedRemainingWeightG != "595.500" {
		t.Fatalf("RecordSpoolMeasurement()=%#v,%v", value, err)
	}
}
