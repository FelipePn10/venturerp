package validation

import "testing"

func TestUFDoCEP(t *testing.T) {
	casos := map[string]string{
		"01310-100": "SP", "80010-000": "PR", "88010-000": "SC", "90010-000": "RS",
		"70040-010": "DF", "74000-000": "GO", "73700-000": "GO", "69301-000": "RR",
		"69900-000": "AC", "76801-000": "RO", "68900-000": "AP", "29000-000": "ES",
		"1234": "", "00000-000": "",
	}
	for cep, quer := range casos {
		if got := UFDoCEP(cep); got != quer {
			t.Errorf("UFDoCEP(%s) = %q, queria %q", cep, got, quer)
		}
	}
}
