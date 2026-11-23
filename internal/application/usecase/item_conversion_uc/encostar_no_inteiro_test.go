package item_conversion_uc

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestEncostarNoInteiro(t *testing.T) {
	for in, want := range map[string]string{
		"1.0000014": "1", "0.9999996": "1", "1.5": "1.5", "0.000004": "0.000004", "2.00002": "2.00002",
	} {
		if got := encostarNoInteiro(decimal.RequireFromString(in)); !got.Equal(decimal.RequireFromString(want)) {
			t.Errorf("%s → %s, quer %s", in, got, want)
		}
	}
}
