package apiclient

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Material struct {
	ID                               string  `json:"id"`
	Manufacturer                     string  `json:"manufacturer"`
	Name                             string  `json:"name"`
	MaterialType                     string  `json:"material_type"`
	ColorName                        string  `json:"color_name"`
	ColorHex                         *string `json:"color_hex"`
	NominalDensity                   string  `json:"nominal_density"`
	DefaultReplacementCostPerKgCents int64   `json:"default_replacement_cost_per_kg_cents"`
	Notes                            string  `json:"notes"`
	CreatedAt                        string  `json:"created_at"`
	UpdatedAt                        string  `json:"updated_at"`
}
type Spool struct {
	ID                        string  `json:"id"`
	Code                      string  `json:"code"`
	MaterialID                string  `json:"material_id"`
	NominalNetWeightG         string  `json:"nominal_net_weight_g"`
	TareWeightG               string  `json:"tare_weight_g"`
	GrossWeightAtOpenG        *string `json:"gross_weight_at_open_g"`
	CurrentRemainingWeightG   *string `json:"current_remaining_weight_g"`
	PurchaseCostCents         int64   `json:"purchase_cost_cents"`
	ReplacementCostPerKgCents int64   `json:"replacement_cost_per_kg_cents"`
	OpenedAt                  *string `json:"opened_at"`
	LastWeighedAt             *string `json:"last_weighed_at"`
	LastDriedAt               *string `json:"last_dried_at"`
	StorageLocation           string  `json:"storage_location"`
	StorageStatus             string  `json:"storage_status"`
	LotNumber                 string  `json:"lot_number"`
	Status                    string  `json:"status"`
	CreatedAt                 string  `json:"created_at"`
	UpdatedAt                 string  `json:"updated_at"`
}
type SpoolMeasurement struct {
	ID                      string `json:"id"`
	SpoolID                 string `json:"spool_id"`
	MeasuredAt              string `json:"measured_at"`
	GrossWeightG            string `json:"gross_weight_g"`
	DerivedRemainingWeightG string `json:"derived_remaining_weight_g"`
	Source                  string `json:"source"`
	Notes                   string `json:"notes"`
	RecordedBy              string `json:"recorded_by"`
	CreatedAt               string `json:"created_at"`
}
type MeasurementInput struct {
	MeasuredAt   string `json:"measured_at"`
	GrossWeightG string `json:"gross_weight_g"`
	Source       string `json:"source"`
	Notes        string `json:"notes"`
}

func (client *Client) ListMaterials(ctx context.Context, token string) ([]Material, error) {
	var result struct {
		Materials []Material `json:"materials"`
	}
	if err := client.catalogJSON(ctx, http.MethodGet, "/api/v1/inventory/materials", token, nil, &result); err != nil {
		return nil, err
	}
	if result.Materials == nil {
		return nil, invalidResponseError("Server returned invalid materials", nil)
	}
	for _, value := range result.Materials {
		if !validMaterial(value) {
			return nil, invalidResponseError("Server returned invalid materials", nil)
		}
	}
	return result.Materials, nil
}
func (client *Client) ListSpools(ctx context.Context, token string) ([]Spool, error) {
	var result struct {
		Spools []Spool `json:"spools"`
	}
	if err := client.catalogJSON(ctx, http.MethodGet, "/api/v1/inventory/spools", token, nil, &result); err != nil {
		return nil, err
	}
	if result.Spools == nil {
		return nil, invalidResponseError("Server returned invalid spools", nil)
	}
	for _, value := range result.Spools {
		if !validSpool(value) {
			return nil, invalidResponseError("Server returned invalid spools", nil)
		}
	}
	return result.Spools, nil
}
func (client *Client) ListSpoolMeasurements(ctx context.Context, token, spoolID string) ([]SpoolMeasurement, error) {
	var result struct {
		Measurements []SpoolMeasurement `json:"measurements"`
	}
	path := "/api/v1/inventory/spools/" + url.PathEscape(strings.TrimSpace(spoolID)) + "/measurements"
	if err := client.catalogJSON(ctx, http.MethodGet, path, token, nil, &result); err != nil {
		return nil, err
	}
	if result.Measurements == nil {
		return nil, invalidResponseError("Server returned invalid measurements", nil)
	}
	for _, value := range result.Measurements {
		if !validMeasurement(value) {
			return nil, invalidResponseError("Server returned invalid measurements", nil)
		}
	}
	return result.Measurements, nil
}
func (client *Client) RecordSpoolMeasurement(ctx context.Context, token, spoolID string, input MeasurementInput) (SpoolMeasurement, error) {
	var result SpoolMeasurement
	path := "/api/v1/inventory/spools/" + url.PathEscape(strings.TrimSpace(spoolID)) + "/measurements"
	if err := client.catalogJSON(ctx, http.MethodPost, path, token, input, &result); err != nil {
		return SpoolMeasurement{}, err
	}
	if !validMeasurement(result) {
		return SpoolMeasurement{}, invalidResponseError("Server returned invalid measurement", nil)
	}
	return result, nil
}
func validMaterial(value Material) bool {
	return strings.TrimSpace(value.ID) != "" && strings.TrimSpace(value.Name) != "" && validTime(value.CreatedAt) && validTime(value.UpdatedAt)
}
func validSpool(value Spool) bool {
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.Code) == "" || strings.TrimSpace(value.MaterialID) == "" || !validTime(value.CreatedAt) || !validTime(value.UpdatedAt) {
		return false
	}
	switch value.Status {
	case "sealed", "open", "stored", "drying", "empty", "retired":
		return true
	}
	return false
}
func validMeasurement(value SpoolMeasurement) bool {
	return strings.TrimSpace(value.ID) != "" && strings.TrimSpace(value.SpoolID) != "" && strings.TrimSpace(value.GrossWeightG) != "" && strings.TrimSpace(value.DerivedRemainingWeightG) != "" && validTime(value.MeasuredAt) && validTime(value.CreatedAt)
}
func validTime(value string) bool { _, err := time.Parse(time.RFC3339, value); return err == nil }
