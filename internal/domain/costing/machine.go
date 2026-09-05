package costing

import "math/big"

type MachineInputs struct {
	AcquisitionCostCents           int64
	ResidualValueCents             int64
	UsefulLifeHours                string
	MaintenanceReservePerHourCents int64
}

type MachineRate struct {
	DepreciationPerHour *big.Rat
	TotalPerHour        *big.Rat
}

// CalculateMachineRate preserves fractional cents for downstream duration calculations.
// Energy, labor and machine-rate overrides are outside this formula (PRD 17.3).
func CalculateMachineRate(input MachineInputs) (MachineRate, error) {
	hours, err := Decimal(input.UsefulLifeHours)
	if err != nil || hours.Sign() <= 0 || input.AcquisitionCostCents < 0 || input.ResidualValueCents < 0 || input.ResidualValueCents > input.AcquisitionCostCents || input.MaintenanceReservePerHourCents < 0 {
		return MachineRate{}, ErrInvalidInput
	}
	depreciation := new(big.Rat).Quo(new(big.Rat).SetInt64(input.AcquisitionCostCents-input.ResidualValueCents), hours)
	total := new(big.Rat).Add(depreciation, new(big.Rat).SetInt64(input.MaintenanceReservePerHourCents))
	return MachineRate{DepreciationPerHour: depreciation, TotalPerHour: total}, nil
}
