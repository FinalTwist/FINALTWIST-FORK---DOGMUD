package characters

import "testing"

// CanAffordCostFloat must give exactly ApplyCostFloatOrRefuse's verdict and
// change nothing. Movement parity 4b quotes a step before issuing it; a quote
// that disagreed with the charge would issue steps the charge then refuses.
func TestCanAffordCostFloat_AgreesWithTheChargeAndMutatesNothing(t *testing.T) {
	cases := []struct {
		name    string
		stamina int
		carry   float64
		amount  float64
	}{
		{"sub-one step on an empty pool banks, so it is affordable", 0, 0, 0.55},
		{"a whole point on an empty pool is not", 0, 0, 1.4},
		{"a whole point with one in the pool is", 1, 0, 1.4},
		{"carry tips a sub-one step over a whole point", 0, 0.6, 0.55},
		{"NaN is free", 0, 0, nanForTest()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			build := func() *Character {
				c := &Character{Stamina: tc.stamina}
				c.StaminaMax.Value = 100
				if tc.carry > 0 {
					c.costCarry = map[Pool]float64{PoolStamina: tc.carry}
				}
				return c
			}
			quoted := build()
			got := quoted.CanAffordCostFloat(PoolStamina, tc.amount)
			if quoted.Stamina != tc.stamina || quoted.costCarry[PoolStamina] != tc.carry {
				t.Fatalf("the quote mutated the character: stamina %d carry %v", quoted.Stamina, quoted.costCarry[PoolStamina])
			}
			want := build().ApplyCostFloatOrRefuse(PoolStamina, tc.amount)
			if got != want {
				t.Fatalf("CanAffordCostFloat = %v, ApplyCostFloatOrRefuse = %v", got, want)
			}
		})
	}
}

func nanForTest() float64 {
	zero := 0.0
	return zero / zero
}
