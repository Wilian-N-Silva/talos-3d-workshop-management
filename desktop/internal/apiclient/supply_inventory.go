package apiclient

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

type Supply struct {
	ID                       string  `json:"id"`
	Name                     string  `json:"name"`
	SKU                      *string `json:"sku"`
	Unit                     string  `json:"unit"`
	CurrentQuantity          string  `json:"current_quantity"`
	ReplacementUnitCostCents int64   `json:"replacement_unit_cost_cents"`
	MinimumQuantity          string  `json:"minimum_quantity"`
	Notes                    string  `json:"notes"`
	CreatedAt                string  `json:"created_at"`
	UpdatedAt                string  `json:"updated_at"`
}

type SupplyInput struct {
	Name                     string  `json:"name"`
	SKU                      *string `json:"sku"`
	Unit                     string  `json:"unit"`
	ReplacementUnitCostCents int64   `json:"replacement_unit_cost_cents"`
	MinimumQuantity          string  `json:"minimum_quantity"`
	Notes                    string  `json:"notes"`
}

type SupplyMovement struct {
	ID            string  `json:"id"`
	SupplyID      string  `json:"supply_id"`
	Type          string  `json:"type"`
	Quantity      string  `json:"quantity"`
	UnitCostCents *int64  `json:"unit_cost_cents"`
	ReferenceType *string `json:"reference_type"`
	ReferenceID   *string `json:"reference_id"`
	OccurredAt    string  `json:"occurred_at"`
	RecordedBy    string  `json:"recorded_by"`
	Notes         string  `json:"notes"`
	CreatedAt     string  `json:"created_at"`
}

type SupplyMovementInput struct {
	Type          string  `json:"type"`
	Quantity      string  `json:"quantity"`
	UnitCostCents *int64  `json:"unit_cost_cents"`
	ReferenceType *string `json:"reference_type"`
	ReferenceID   *string `json:"reference_id"`
	OccurredAt    string  `json:"occurred_at"`
	Notes         string  `json:"notes"`
}

type LowInventory struct {
	SpoolThresholdG string   `json:"spool_threshold_g"`
	Spools          []Spool  `json:"spools"`
	Supplies        []Supply `json:"supplies"`
}

func (client *Client) ListSupplies(ctx context.Context, token string) ([]Supply, error) {
	var result struct {
		Supplies []Supply `json:"supplies"`
	}
	if err := client.catalogJSON(ctx, http.MethodGet, "/api/v1/inventory/supplies", token, nil, &result); err != nil {
		return nil, err
	}
	if result.Supplies == nil {
		return nil, invalidResponseError("Server returned invalid supplies", nil)
	}
	for _, value := range result.Supplies {
		if !validSupply(value) {
			return nil, invalidResponseError("Server returned invalid supplies", nil)
		}
	}
	return result.Supplies, nil
}

func (client *Client) CreateSupply(ctx context.Context, token string, input SupplyInput) (Supply, error) {
	var result Supply
	if err := client.catalogJSON(ctx, http.MethodPost, "/api/v1/inventory/supplies", token, input, &result); err != nil {
		return Supply{}, err
	}
	if !validSupply(result) {
		return Supply{}, invalidResponseError("Server returned invalid supply", nil)
	}
	return result, nil
}

func (client *Client) ListSupplyMovements(ctx context.Context, token, supplyID string) ([]SupplyMovement, error) {
	var result struct {
		Movements []SupplyMovement `json:"movements"`
	}
	path := "/api/v1/inventory/supplies/" + url.PathEscape(strings.TrimSpace(supplyID)) + "/movements"
	if err := client.catalogJSON(ctx, http.MethodGet, path, token, nil, &result); err != nil {
		return nil, err
	}
	if result.Movements == nil {
		return nil, invalidResponseError("Server returned invalid supply movements", nil)
	}
	for _, value := range result.Movements {
		if !validSupplyMovement(value) {
			return nil, invalidResponseError("Server returned invalid supply movements", nil)
		}
	}
	return result.Movements, nil
}

func (client *Client) RecordSupplyMovement(ctx context.Context, token, supplyID string, input SupplyMovementInput) (SupplyMovement, error) {
	var result SupplyMovement
	path := "/api/v1/inventory/supplies/" + url.PathEscape(strings.TrimSpace(supplyID)) + "/movements"
	if err := client.catalogJSON(ctx, http.MethodPost, path, token, input, &result); err != nil {
		return SupplyMovement{}, err
	}
	if !validSupplyMovement(result) {
		return SupplyMovement{}, invalidResponseError("Server returned invalid supply movement", nil)
	}
	return result, nil
}

func (client *Client) ListLowInventory(ctx context.Context, token, spoolThresholdG string) (LowInventory, error) {
	path := "/api/v1/inventory/low-stock?spool_threshold_g=" + url.QueryEscape(strings.TrimSpace(spoolThresholdG))
	var result LowInventory
	if err := client.catalogJSON(ctx, http.MethodGet, path, token, nil, &result); err != nil {
		return LowInventory{}, err
	}
	if strings.TrimSpace(result.SpoolThresholdG) == "" || result.Spools == nil || result.Supplies == nil {
		return LowInventory{}, invalidResponseError("Server returned invalid low inventory", nil)
	}
	for _, value := range result.Spools {
		if !validSpool(value) {
			return LowInventory{}, invalidResponseError("Server returned invalid low inventory", nil)
		}
	}
	for _, value := range result.Supplies {
		if !validSupply(value) {
			return LowInventory{}, invalidResponseError("Server returned invalid low inventory", nil)
		}
	}
	return result, nil
}

func validSupply(value Supply) bool {
	return strings.TrimSpace(value.ID) != "" && strings.TrimSpace(value.Name) != "" && strings.TrimSpace(value.Unit) != "" && strings.TrimSpace(value.CurrentQuantity) != "" && strings.TrimSpace(value.MinimumQuantity) != "" && validTime(value.CreatedAt) && validTime(value.UpdatedAt)
}

func validSupplyMovement(value SupplyMovement) bool {
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.SupplyID) == "" || strings.TrimSpace(value.Quantity) == "" || !validTime(value.OccurredAt) || !validTime(value.CreatedAt) {
		return false
	}
	switch value.Type {
	case "purchase", "consume", "adjustment", "return", "discard":
		return true
	}
	return false
}
