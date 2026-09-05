package apiclient

import (
	"context"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

var laborHoursPattern = regexp.MustCompile(`^[0-9]{1,15}(\.[0-9]{1,10})?$`)

const laborRatesPath = "/api/v1/costing/labor-rates"

// Monetary values cross the JavaScript bridge as strings to retain all int64 cents.
type LaborRate struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	ActivityType        string `json:"activity_type"`
	CostHourlyRateCents string `json:"cost_hourly_rate_cents"`
	Active              bool   `json:"active"`
}
type LaborRateInput struct {
	Name                string `json:"name"`
	ActivityType        string `json:"activity_type"`
	CostHourlyRateCents string `json:"cost_hourly_rate_cents"`
	Active              bool   `json:"active"`
}
type LaborAssumptions struct {
	TargetMonthlyCompensationCents string `json:"target_monthly_compensation_cents"`
	MonthlyLaborOverheadCents      string `json:"monthly_labor_overhead_cents"`
	AvailableHoursPerMonth         string `json:"available_hours_per_month"`
	ProductiveUtilizationBPS       int64  `json:"productive_utilization_bps"`
}
type LaborSuggestion struct {
	ProductiveHours         string `json:"productive_hours"`
	InternalHourlyCostCents string `json:"internal_hourly_cost_cents"`
}
type laborRateWire struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ActivityType string `json:"activity_type"`
	Cents        *int64 `json:"cost_hourly_rate_cents"`
	Active       bool   `json:"active"`
}

func (r laborRateWire) safe() (LaborRate, error) {
	if r.ID == "" || strings.TrimSpace(r.Name) == "" || r.Cents == nil || *r.Cents < 0 || r.ActivityType == "" {
		return LaborRate{}, invalidResponseError("Invalid labor rate", nil)
	}
	return LaborRate{r.ID, r.Name, r.ActivityType, strconv.FormatInt(*r.Cents, 10), r.Active}, nil
}
func (client *Client) ListLaborRates(ctx context.Context, token string) ([]LaborRate, error) {
	var result struct {
		Rates []laborRateWire `json:"rates"`
	}
	if err := client.catalogJSON(ctx, http.MethodGet, laborRatesPath, token, nil, &result); err != nil {
		return nil, err
	}
	if result.Rates == nil {
		return nil, invalidResponseError("Invalid labor rates", nil)
	}
	rates := make([]LaborRate, 0, len(result.Rates))
	for _, r := range result.Rates {
		safe, err := r.safe()
		if err != nil {
			return nil, err
		}
		rates = append(rates, safe)
	}
	return rates, nil
}
func (client *Client) SaveLaborRate(ctx context.Context, token, id string, input LaborRateInput) (LaborRate, error) {
	cents, err := laborCents(input.CostHourlyRateCents)
	if err != nil {
		return LaborRate{}, err
	}
	method, path := http.MethodPost, laborRatesPath
	if id != "" {
		method = http.MethodPut
		path += "/" + url.PathEscape(id)
	}
	body := struct {
		Name     string `json:"name"`
		Activity string `json:"activity_type"`
		Cents    int64  `json:"cost_hourly_rate_cents"`
		Active   bool   `json:"active"`
	}{input.Name, input.ActivityType, cents, input.Active}
	var result laborRateWire
	if err := client.catalogJSON(ctx, method, path, token, body, &result); err != nil {
		return LaborRate{}, err
	}
	return result.safe()
}
func (client *Client) SuggestLaborRate(ctx context.Context, token string, input LaborAssumptions) (LaborSuggestion, error) {
	compensation, err := laborCents(input.TargetMonthlyCompensationCents)
	if err != nil {
		return LaborSuggestion{}, err
	}
	overhead, err := laborCents(input.MonthlyLaborOverheadCents)
	if err != nil {
		return LaborSuggestion{}, err
	}
	body := struct {
		Compensation int64  `json:"target_monthly_compensation_cents"`
		Overhead     int64  `json:"monthly_labor_overhead_cents"`
		Hours        string `json:"available_hours_per_month"`
		Utilization  int64  `json:"productive_utilization_bps"`
	}{compensation, overhead, input.AvailableHoursPerMonth, input.ProductiveUtilizationBPS}
	var result struct {
		Hours string `json:"productive_hours"`
		Cents *int64 `json:"internal_hourly_cost_cents"`
	}
	if err := client.catalogJSON(ctx, http.MethodPost, laborRatesPath+"/suggestion", token, body, &result); err != nil {
		return LaborSuggestion{}, err
	}
	hours, validHours := new(big.Rat).SetString(result.Hours)
	if !laborHoursPattern.MatchString(result.Hours) || !validHours || hours.Sign() <= 0 || result.Cents == nil || *result.Cents < 0 {
		return LaborSuggestion{}, invalidResponseError("Invalid labor suggestion", nil)
	}
	return LaborSuggestion{result.Hours, strconv.FormatInt(*result.Cents, 10)}, nil
}
func laborCents(value string) (int64, error) {
	cents, err := strconv.ParseInt(value, 10, 64)
	if err != nil || cents < 0 {
		return 0, invalidResponseError("Informe um valor monetário válido em centavos", nil)
	}
	return cents, nil
}
