package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const jobJSON = `{"id":"11111111-1111-4111-8111-111111111111","code":"JOB-1","catalog_item_id":"22222222-2222-4222-8222-222222222222","design_version_id":"33333333-3333-4333-8333-333333333333","printer_id":"44444444-4444-4444-8444-444444444444","order_item_id":null,"purpose":"internal","status":"draft","planned_quantity":1,"good_quantity":0,"scrap_quantity":0,"quality_status":"pending","created_at":"2026-09-04T12:00:00Z","updated_at":"2026-09-04T12:00:00Z"}`
const usageJSON = `{"id":"55555555-5555-4555-8555-555555555555","print_job_id":"11111111-1111-4111-8111-111111111111","material_id":"66666666-6666-4666-8666-666666666666","spool_id":"77777777-7777-4777-8777-777777777777","role":"model","planned_grams":"12.5","actual_grams":"11.75","planned_meters":null,"actual_meters":null,"measurement_source":"spool_weight_delta","historical_material_cost_cents":125,"replacement_material_cost_cents":150,"created_at":"2026-09-04T12:00:00Z","updated_at":"2026-09-04T12:00:00Z"}`

func TestClientJobMaterialUsageFlowKeepsBearerAndExactValues(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		if request.Header.Get("Authorization") != "Bearer secure-token" {
			t.Fatalf("authorization=%q", request.Header.Get("Authorization"))
		}
		switch requests {
		case 1:
			if request.Method != http.MethodGet || request.URL.Path != jobsPath {
				t.Fatalf("jobs request=%s %s", request.Method, request.URL.Path)
			}
			_, _ = response.Write([]byte(`{"jobs":[` + jobJSON + `]}`))
		case 2:
			if request.Method != http.MethodGet || request.URL.Path != jobsPath+"/11111111-1111-4111-8111-111111111111/materials" {
				t.Fatalf("usage list request=%s %s", request.Method, request.URL.Path)
			}
			_, _ = response.Write([]byte(`{"items":[` + usageJSON + `],"total_planned_grams":"12.5","total_actual_grams":"11.75"}`))
		case 3, 4:
			wantMethod := map[int]string{3: http.MethodPost, 4: http.MethodPut}[requests]
			if request.Method != wantMethod || request.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("write request=%s content-type=%q", request.Method, request.Header.Get("Content-Type"))
			}
			var input JobMaterialUsageInput
			if err := json.NewDecoder(request.Body).Decode(&input); err != nil || input.PlannedGrams != "12.5" || input.ActualGrams == nil || *input.ActualGrams != "11.75" || input.MeasurementSource != "spool_weight_delta" {
				t.Fatalf("input=%#v error=%v", input, err)
			}
			_, _ = response.Write([]byte(usageJSON))
		case 5:
			if request.Method != http.MethodDelete || request.URL.Path != jobsPath+"/11111111-1111-4111-8111-111111111111/materials/55555555-5555-4555-8555-555555555555" {
				t.Fatalf("delete request=%s %s", request.Method, request.URL.Path)
			}
			response.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	client, _ := New(server.URL, "1.0.0")
	jobs, err := client.ListJobs(context.Background(), "secure-token")
	if err != nil || len(jobs) != 1 || jobs[0].Code != "JOB-1" {
		t.Fatalf("ListJobs()=%#v,%v", jobs, err)
	}
	summary, err := client.ListJobMaterialUsage(context.Background(), "secure-token", jobs[0].ID)
	if err != nil || summary.TotalPlannedGrams != "12.5" || summary.TotalActualGrams != "11.75" || summary.Items[0].MeasurementSource != "spool_weight_delta" {
		t.Fatalf("ListJobMaterialUsage()=%#v,%v", summary, err)
	}
	actual := "11.75"
	input := JobMaterialUsageInput{SpoolID: "77777777-7777-4777-8777-777777777777", Role: "model", PlannedGrams: "12.5", ActualGrams: &actual, MeasurementSource: "spool_weight_delta"}
	if _, err := client.CreateJobMaterialUsage(context.Background(), "secure-token", jobs[0].ID, input); err != nil {
		t.Fatalf("CreateJobMaterialUsage()=%v", err)
	}
	if _, err := client.UpdateJobMaterialUsage(context.Background(), "secure-token", jobs[0].ID, "55555555-5555-4555-8555-555555555555", input); err != nil {
		t.Fatalf("UpdateJobMaterialUsage()=%v", err)
	}
	if err := client.DeleteJobMaterialUsage(context.Background(), "secure-token", jobs[0].ID, "55555555-5555-4555-8555-555555555555"); err != nil {
		t.Fatalf("DeleteJobMaterialUsage()=%v", err)
	}
}
