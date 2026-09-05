package apiclient

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

const jobsPath = "/api/v1/jobs"

type Job struct {
	ID              string  `json:"id"`
	Code            string  `json:"code"`
	CatalogItemID   string  `json:"catalog_item_id"`
	DesignVersionID string  `json:"design_version_id"`
	PrinterID       string  `json:"printer_id"`
	OrderItemID     *string `json:"order_item_id"`
	Purpose         string  `json:"purpose"`
	Status          string  `json:"status"`
	PlannedQuantity int     `json:"planned_quantity"`
	GoodQuantity    int     `json:"good_quantity"`
	ScrapQuantity   int     `json:"scrap_quantity"`
	QualityStatus   string  `json:"quality_status"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type JobMaterialUsage struct {
	ID                           string  `json:"id"`
	PrintJobID                   string  `json:"print_job_id"`
	MaterialID                   string  `json:"material_id"`
	SpoolID                      string  `json:"spool_id"`
	Role                         string  `json:"role"`
	PlannedGrams                 string  `json:"planned_grams"`
	ActualGrams                  *string `json:"actual_grams"`
	PlannedMeters                *string `json:"planned_meters"`
	ActualMeters                 *string `json:"actual_meters"`
	MeasurementSource            string  `json:"measurement_source"`
	HistoricalMaterialCostCents  *int64  `json:"historical_material_cost_cents"`
	ReplacementMaterialCostCents *int64  `json:"replacement_material_cost_cents"`
	CreatedAt                    string  `json:"created_at"`
	UpdatedAt                    string  `json:"updated_at"`
}

type JobMaterialUsageInput struct {
	SpoolID                      string  `json:"spool_id"`
	Role                         string  `json:"role"`
	PlannedGrams                 string  `json:"planned_grams"`
	ActualGrams                  *string `json:"actual_grams"`
	PlannedMeters                *string `json:"planned_meters"`
	ActualMeters                 *string `json:"actual_meters"`
	MeasurementSource            string  `json:"measurement_source"`
	HistoricalMaterialCostCents  *int64  `json:"historical_material_cost_cents"`
	ReplacementMaterialCostCents *int64  `json:"replacement_material_cost_cents"`
}

type JobMaterialUsageSummary struct {
	Items             []JobMaterialUsage `json:"items"`
	TotalPlannedGrams string             `json:"total_planned_grams"`
	TotalActualGrams  string             `json:"total_actual_grams"`
}

func (client *Client) ListJobs(ctx context.Context, token string) ([]Job, error) {
	var result struct {
		Jobs []Job `json:"jobs"`
	}
	if err := client.catalogJSON(ctx, http.MethodGet, jobsPath, token, nil, &result); err != nil {
		return nil, err
	}
	if result.Jobs == nil {
		return nil, invalidResponseError("Server returned invalid print jobs", nil)
	}
	for _, value := range result.Jobs {
		if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.Code) == "" || !validJobStatus(value.Status) || !validTime(value.CreatedAt) || !validTime(value.UpdatedAt) {
			return nil, invalidResponseError("Server returned invalid print jobs", nil)
		}
	}
	return result.Jobs, nil
}

func (client *Client) ListJobMaterialUsage(ctx context.Context, token, jobID string) (JobMaterialUsageSummary, error) {
	var result JobMaterialUsageSummary
	path := jobMaterialUsagePath(jobID)
	if err := client.catalogJSON(ctx, http.MethodGet, path, token, nil, &result); err != nil {
		return JobMaterialUsageSummary{}, err
	}
	if result.Items == nil || strings.TrimSpace(result.TotalPlannedGrams) == "" || strings.TrimSpace(result.TotalActualGrams) == "" {
		return JobMaterialUsageSummary{}, invalidResponseError("Server returned invalid job material usage", nil)
	}
	for _, value := range result.Items {
		if !validJobMaterialUsage(value) {
			return JobMaterialUsageSummary{}, invalidResponseError("Server returned invalid job material usage", nil)
		}
	}
	return result, nil
}

func (client *Client) CreateJobMaterialUsage(ctx context.Context, token, jobID string, input JobMaterialUsageInput) (JobMaterialUsage, error) {
	var result JobMaterialUsage
	if err := client.catalogJSON(ctx, http.MethodPost, jobMaterialUsagePath(jobID), token, input, &result); err != nil {
		return JobMaterialUsage{}, err
	}
	if !validJobMaterialUsage(result) {
		return JobMaterialUsage{}, invalidResponseError("Server returned invalid job material usage", nil)
	}
	return result, nil
}

func (client *Client) UpdateJobMaterialUsage(ctx context.Context, token, jobID, usageID string, input JobMaterialUsageInput) (JobMaterialUsage, error) {
	var result JobMaterialUsage
	path := jobMaterialUsagePath(jobID) + "/" + url.PathEscape(strings.TrimSpace(usageID))
	if err := client.catalogJSON(ctx, http.MethodPut, path, token, input, &result); err != nil {
		return JobMaterialUsage{}, err
	}
	if !validJobMaterialUsage(result) {
		return JobMaterialUsage{}, invalidResponseError("Server returned invalid job material usage", nil)
	}
	return result, nil
}

func (client *Client) DeleteJobMaterialUsage(ctx context.Context, token, jobID, usageID string) error {
	path := jobMaterialUsagePath(jobID) + "/" + url.PathEscape(strings.TrimSpace(usageID))
	return client.catalogJSON(ctx, http.MethodDelete, path, token, nil, nil)
}

func jobMaterialUsagePath(jobID string) string {
	return jobsPath + "/" + url.PathEscape(strings.TrimSpace(jobID)) + "/materials"
}

func validJobMaterialUsage(value JobMaterialUsage) bool {
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.PrintJobID) == "" || strings.TrimSpace(value.MaterialID) == "" || strings.TrimSpace(value.SpoolID) == "" || strings.TrimSpace(value.PlannedGrams) == "" || !validTime(value.CreatedAt) || !validTime(value.UpdatedAt) {
		return false
	}
	return (value.Role == "model" || value.Role == "support" || value.Role == "purge" || value.Role == "other") &&
		(value.MeasurementSource == "slicer" || value.MeasurementSource == "spool_weight_delta" || value.MeasurementSource == "manual" || value.MeasurementSource == "printer" || value.MeasurementSource == "estimated")
}

func validJobStatus(value string) bool {
	switch value {
	case "draft", "prepared", "printing", "awaiting_review", "completed", "failed", "cancelled":
		return true
	}
	return false
}
