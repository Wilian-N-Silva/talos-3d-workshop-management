package apiclient

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

type CatalogBOMItem struct {
	ID              string `json:"id"`
	CatalogItemID   string `json:"catalog_item_id"`
	SupplyID        string `json:"supply_id"`
	QuantityPerUnit string `json:"quantity_per_unit"`
	WastePercent    string `json:"waste_percent"`
	Notes           string `json:"notes"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type CatalogBOMInput struct {
	SupplyID        string `json:"supply_id"`
	QuantityPerUnit string `json:"quantity_per_unit"`
	WastePercent    string `json:"waste_percent"`
	Notes           string `json:"notes"`
}

type CatalogBOMPreviewLine struct {
	CatalogBOMItem
	SupplyName                       string `json:"supply_name"`
	SupplyUnit                       string `json:"supply_unit"`
	ReplacementUnitCostCents         int64  `json:"replacement_unit_cost_cents"`
	EffectiveQuantityPerUnit         string `json:"effective_quantity_per_unit"`
	ExactReplacementCostCentsPerUnit string `json:"exact_replacement_cost_cents_per_unit"`
}

type CatalogBOMPreview struct {
	Items                          []CatalogBOMPreviewLine `json:"items"`
	ExactTotalReplacementCostCents string                  `json:"exact_total_replacement_cost_cents"`
	RoundingApplied                bool                    `json:"rounding_applied"`
}

func (client *Client) GetCatalogBOM(ctx context.Context, token, itemID string) (CatalogBOMPreview, error) {
	var result CatalogBOMPreview
	path := catalogBOMPath(itemID)
	if err := client.catalogJSON(ctx, http.MethodGet, path, token, nil, &result); err != nil {
		return CatalogBOMPreview{}, err
	}
	if result.Items == nil || strings.TrimSpace(result.ExactTotalReplacementCostCents) == "" || result.RoundingApplied {
		return CatalogBOMPreview{}, invalidResponseError("Server returned invalid catalog BOM", nil)
	}
	for _, item := range result.Items {
		if !validCatalogBOMItem(item.CatalogBOMItem) || strings.TrimSpace(item.SupplyName) == "" || strings.TrimSpace(item.SupplyUnit) == "" || strings.TrimSpace(item.EffectiveQuantityPerUnit) == "" || strings.TrimSpace(item.ExactReplacementCostCentsPerUnit) == "" || item.ReplacementUnitCostCents < 0 {
			return CatalogBOMPreview{}, invalidResponseError("Server returned invalid catalog BOM", nil)
		}
	}
	return result, nil
}

func (client *Client) CreateCatalogBOMItem(ctx context.Context, token, itemID string, input CatalogBOMInput) (CatalogBOMItem, error) {
	var result CatalogBOMItem
	if err := client.catalogJSON(ctx, http.MethodPost, catalogBOMPath(itemID), token, input, &result); err != nil {
		return CatalogBOMItem{}, err
	}
	if !validCatalogBOMItem(result) {
		return CatalogBOMItem{}, invalidResponseError("Server returned invalid catalog BOM item", nil)
	}
	return result, nil
}

func (client *Client) UpdateCatalogBOMItem(ctx context.Context, token, itemID, bomItemID string, input CatalogBOMInput) (CatalogBOMItem, error) {
	var result CatalogBOMItem
	path := catalogBOMPath(itemID) + "/" + url.PathEscape(strings.TrimSpace(bomItemID))
	if err := client.catalogJSON(ctx, http.MethodPut, path, token, input, &result); err != nil {
		return CatalogBOMItem{}, err
	}
	if !validCatalogBOMItem(result) {
		return CatalogBOMItem{}, invalidResponseError("Server returned invalid catalog BOM item", nil)
	}
	return result, nil
}

func (client *Client) DeleteCatalogBOMItem(ctx context.Context, token, itemID, bomItemID string) error {
	path := catalogBOMPath(itemID) + "/" + url.PathEscape(strings.TrimSpace(bomItemID))
	return client.catalogJSON(ctx, http.MethodDelete, path, token, nil, nil)
}

func catalogBOMPath(itemID string) string {
	return catalogItemsPath + "/" + url.PathEscape(strings.TrimSpace(itemID)) + "/bom"
}

func validCatalogBOMItem(item CatalogBOMItem) bool {
	return strings.TrimSpace(item.ID) != "" && strings.TrimSpace(item.CatalogItemID) != "" && strings.TrimSpace(item.SupplyID) != "" && strings.TrimSpace(item.QuantityPerUnit) != "" && strings.TrimSpace(item.WastePercent) != "" && validTime(item.CreatedAt) && validTime(item.UpdatedAt)
}
