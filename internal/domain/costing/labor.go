package costing

import "math/big"

type LaborAssumptions struct {
	TargetMonthlyCompensationCents int64  `json:"target_monthly_compensation_cents"`
	MonthlyLaborOverheadCents      int64  `json:"monthly_labor_overhead_cents"`
	AvailableHoursPerMonth         string `json:"available_hours_per_month"`
	ProductiveUtilizationBPS       int64  `json:"productive_utilization_bps"`
}

type LaborSuggestion struct {
	// ProductiveHours is exact: bounded decimal hours times integer basis points.
	ProductiveHours         string `json:"productive_hours"`
	InternalHourlyCostCents int64  `json:"internal_hourly_cost_cents"`
}

// SuggestLaborRate follows PRD 17.4 and rounds only the final suggested rate.
func SuggestLaborRate(input LaborAssumptions) (LaborSuggestion, error) {
	hours, err := Decimal(input.AvailableHoursPerMonth)
	if err != nil || hours.Sign() <= 0 || input.TargetMonthlyCompensationCents < 0 || input.MonthlyLaborOverheadCents < 0 || input.ProductiveUtilizationBPS <= 0 || input.ProductiveUtilizationBPS > 10000 {
		return LaborSuggestion{}, ErrInvalidInput
	}
	productive := new(big.Rat).Mul(hours, BasisPoints(input.ProductiveUtilizationBPS))
	total := new(big.Rat).Add(new(big.Rat).SetInt64(input.TargetMonthlyCompensationCents), new(big.Rat).SetInt64(input.MonthlyLaborOverheadCents))
	cents, err := RoundCents(new(big.Rat).Quo(total, productive))
	if err != nil {
		return LaborSuggestion{}, err
	}
	// Six decimal places of hours and four of utilization produce at most ten.
	return LaborSuggestion{ProductiveHours: productive.FloatString(10), InternalHourlyCostCents: cents}, nil
}
