package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLaborClientExactMoneyAndExplicitSave(t *testing.T) {
	writes := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing authorization")
		}
		if r.URL.Path == laborRatesPath+"/suggestion" {
			if r.Method != http.MethodPost {
				t.Error(r.Method)
			}
			var body struct {
				Compensation int64 `json:"target_monthly_compensation_cents"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Compensation != 9223372036854775807 {
				t.Error(body, err)
			}
			_, _ = w.Write([]byte(`{"productive_hours":"1.0000000000","internal_hourly_cost_cents":9223372036854775807}`))
			return
		}
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"rates":[]}`))
			return
		}
		writes++
		if r.Method != http.MethodPut || r.URL.Path != laborRatesPath+"/rate-id" {
			t.Error(r.Method, r.URL.Path)
		}
		var body struct {
			Cents int64 `json:"cost_hourly_rate_cents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Cents != 2918 {
			t.Error(body, err)
		}
		_, _ = w.Write([]byte(`{"id":"rate-id","name":"Setup","activity_type":"setup","cost_hourly_rate_cents":2918,"active":true}`))
	}))
	defer server.Close()
	client, _ := New(server.URL, "1.0.0")
	result, err := client.SuggestLaborRate(context.Background(), "test-token", LaborAssumptions{TargetMonthlyCompensationCents: "9223372036854775807", MonthlyLaborOverheadCents: "0", AvailableHoursPerMonth: "1", ProductiveUtilizationBPS: 10000})
	if err != nil || result.InternalHourlyCostCents != "9223372036854775807" || writes != 0 {
		t.Fatal(result, err, writes)
	}
	rates, err := client.ListLaborRates(context.Background(), "test-token")
	if err != nil || len(rates) != 0 {
		t.Fatal(rates, err)
	}
	saved, err := client.SaveLaborRate(context.Background(), "test-token", "rate-id", LaborRateInput{Name: "Setup", ActivityType: "setup", CostHourlyRateCents: "2918", Active: true})
	if err != nil || saved.CostHourlyRateCents != "2918" || writes != 1 {
		t.Fatal(saved, err, writes)
	}
	for _, value := range []string{"9223372036854775808", "-1", "1.5", ""} {
		if _, err := client.SaveLaborRate(context.Background(), "test-token", "", LaborRateInput{CostHourlyRateCents: value}); err == nil {
			t.Fatal("accepted", value)
		}
	}
}

func TestLaborClientRejectsMissingAndInvalidMoney(t *testing.T) {
	for _, body := range []string{`{"productive_hours":"1"}`, `{"productive_hours":"1","internal_hourly_cost_cents":null}`, `{"productive_hours":"0","internal_hourly_cost_cents":1}`, `{"productive_hours":"NaN","internal_hourly_cost_cents":1}`, `{"productive_hours":"1","internal_hourly_cost_cents":1.5}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
		client, _ := New(server.URL, "1.0.0")
		_, err := client.SuggestLaborRate(context.Background(), "token", LaborAssumptions{TargetMonthlyCompensationCents: "1", MonthlyLaborOverheadCents: "0"})
		server.Close()
		if err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
