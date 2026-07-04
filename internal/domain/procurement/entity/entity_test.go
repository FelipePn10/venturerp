package entity

import "testing"

func TestSupplierContractItemRemainingQty(t *testing.T) {
	cases := []struct {
		name                 string
		contracted, consumed float64
		want                 float64
	}{
		{"nothing consumed", 100, 0, 100},
		{"partial", 100, 40, 60},
		{"fully consumed", 100, 100, 0},
		{"over-consumed clamps to zero", 100, 130, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			it := &SupplierContractItem{ContractedQty: c.contracted, ConsumedQty: c.consumed}
			if got := it.RemainingQty(); got != c.want {
				t.Errorf("RemainingQty()=%v want %v", got, c.want)
			}
		})
	}
}
