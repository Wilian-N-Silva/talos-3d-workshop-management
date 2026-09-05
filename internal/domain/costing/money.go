// Package costing implements exact, side-effect-free financial calculations.
package costing

import (
	"errors"
	"math/big"
	"regexp"
)

var ErrInvalidInput = errors.New("invalid costing input")
var ErrOverflow = errors.New("monetary result exceeds integer cents")
var decimalPattern = regexp.MustCompile(`^[0-9]{1,15}(\.[0-9]{1,6})?$`)

// Decimal parses bounded nonnegative decimal input without binary floats.
func Decimal(value string) (*big.Rat, error) {
	if !decimalPattern.MatchString(value) {
		return nil, ErrInvalidInput
	}
	result, ok := new(big.Rat).SetString(value)
	if !ok {
		return nil, ErrInvalidInput
	}
	return result, nil
}

// BasisPoints returns an exact percentage; domain-specific limits belong to callers.
func BasisPoints(value int64) *big.Rat {
	return new(big.Rat).SetFrac(big.NewInt(value), big.NewInt(10000))
}

// RoundCents applies ADR-FIN-001: nearest integer, ties away from zero.
// It does not mutate value and rejects overflow after rounding.
func RoundCents(value *big.Rat) (int64, error) {
	if value == nil {
		return 0, ErrInvalidInput
	}
	magnitude := new(big.Int).Abs(value.Num())
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(magnitude, value.Denom(), remainder)
	if remainder.Lsh(remainder, 1).Cmp(value.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if value.Sign() < 0 {
		quotient.Neg(quotient)
	}
	if !quotient.IsInt64() {
		return 0, ErrOverflow
	}
	return quotient.Int64(), nil
}
