package costing

import (
	"errors"
	"math"
	"math/big"
	"testing"
)

func TestRoundCents(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  int64
	}{
		{"100.5", 101}, {"-100.5", -101}, {"100.49", 100}, {"-100.49", -100}, {"0", 0}, {"0.5", 1}, {"-0.5", -1}, {"1/3", 0}, {"2/3", 1},
		{"9223372036854775807.49", math.MaxInt64}, {"-9223372036854775808.49", math.MinInt64},
	} {
		t.Run(tc.input, func(t *testing.T) {
			r, _ := new(big.Rat).SetString(tc.input)
			before := r.RatString()
			got, err := RoundCents(r)
			if err != nil || got != tc.want || r.RatString() != before {
				t.Fatalf("RoundCents(%s)=%d,%v; mutated=%v", tc.input, got, err, r.RatString() != before)
			}
		})
	}
	for _, input := range []string{"9223372036854775807.5", "-9223372036854775808.5"} {
		r, _ := new(big.Rat).SetString(input)
		if _, err := RoundCents(r); !errors.Is(err, ErrOverflow) {
			t.Fatalf("overflow %s: %v", input, err)
		}
	}
	if _, err := RoundCents(nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatal(err)
	}
}

func TestExactDecimalAndPercentages(t *testing.T) {
	for _, input := range []string{"1/3", "1e3", "NaN", "-1", "", " 1", "1.0000001"} {
		if _, err := Decimal(input); err == nil {
			t.Fatalf("accepted %q", input)
		}
	}
	value, err := Decimal("0.000001")
	if err != nil || value.Cmp(big.NewRat(1, 1000000)) != 0 {
		t.Fatal(value, err)
	}
	for _, tc := range []struct {
		bps  int64
		want string
	}{{7500, "3/4"}, {10000, "1"}, {12500, "5/4"}, {-100, "-1/100"}, {0, "0"}} {
		if got := BasisPoints(tc.bps).RatString(); got != tc.want {
			t.Fatalf("bps %d=%s", tc.bps, got)
		}
	}
}

func TestLaborSuggestion(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input LaborAssumptions
		hours string
		cents int64
	}{
		{"PRD example", LaborAssumptions{300000, 50000, "160", 7500}, "120.0000000000", 2917},
		{"zero cost", LaborAssumptions{0, 0, "160", 7500}, "120.0000000000", 0},
		{"half cent", LaborAssumptions{1, 0, "2", 10000}, "2.0000000000", 1},
		{"fractional hours", LaborAssumptions{100, 0, "0.5", 5000}, "0.2500000000", 400},
		{"sum beyond int64", LaborAssumptions{math.MaxInt64, math.MaxInt64, "2", 10000}, "2.0000000000", math.MaxInt64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SuggestLaborRate(tc.input)
			if err != nil || got.ProductiveHours != tc.hours || got.InternalHourlyCostCents != tc.cents {
				t.Fatalf("got %#v,%v", got, err)
			}
		})
	}
	for _, input := range []LaborAssumptions{{-1, 0, "1", 10000}, {0, -1, "1", 10000}, {1, 0, "0", 10000}, {1, 0, "bad", 10000}, {1, 0, "1", 0}, {1, 0, "1", -1}, {1, 0, "1", 10001}} {
		if _, err := SuggestLaborRate(input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("accepted %#v: %v", input, err)
		}
	}
	if _, err := SuggestLaborRate(LaborAssumptions{math.MaxInt64, 0, "0.1", 10000}); !errors.Is(err, ErrOverflow) {
		t.Fatal(err)
	}
}

func TestMachineRate(t *testing.T) {
	for _, tc := range []struct {
		input               MachineInputs
		depreciation, total string
	}{
		{MachineInputs{100000, 10000, "7000", 2}, "90/7", "104/7"},
		{MachineInputs{100, 100, "1", 2}, "0", "2"},
		{MachineInputs{0, 0, "1", 0}, "0", "0"},
		{MachineInputs{100, 0, "0.5", 0}, "200", "200"},
	} {
		got, err := CalculateMachineRate(tc.input)
		if err != nil || got.DepreciationPerHour.RatString() != tc.depreciation || got.TotalPerHour.RatString() != tc.total {
			t.Fatalf("got %#v,%v", got, err)
		}
	}
	rate, _ := CalculateMachineRate(MachineInputs{100000, 10000, "7000", 2})
	sevenHours := new(big.Rat).Mul(rate.TotalPerHour, big.NewRat(7, 1))
	if cents, _ := RoundCents(sevenHours); cents != 104 {
		t.Fatal(cents)
	}
	rate.TotalPerHour.SetInt64(0)
	if rate.DepreciationPerHour.RatString() != "90/7" {
		t.Fatal("aliased results")
	}
	for _, input := range []MachineInputs{{1, 2, "1", 0}, {-1, 0, "1", 0}, {1, -1, "1", 0}, {1, 0, "1", -1}, {0, 0, "0", 0}, {1, 0, "invalid", 0}} {
		if _, err := CalculateMachineRate(input); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("accepted %#v: %v", input, err)
		}
	}
}
