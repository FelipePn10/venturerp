package validation

import "testing"

func TestSuframaValida(t *testing.T) {
	for in, quer := range map[string]bool{
		"200100122": true, "20.0100.12-2": true, // aceita pela SEFAZ em homologação
		"200100123": false, // recusada (235)
		"20010012":  false, "": false,
	} {
		if got := SuframaValida(in); got != quer {
			t.Errorf("%q → %v, quer %v", in, got, quer)
		}
	}
}
