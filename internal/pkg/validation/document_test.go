package validation_test

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/pkg/validation"
)

func TestValidateCNPJ(t *testing.T) {
	cases := []struct {
		name  string
		cnpj  string
		valid bool
	}{
		{"valid CNPJ formatted", "11.222.333/0001-81", true},
		{"valid CNPJ digits only", "11222333000181", true},
		{"all zeros", "00000000000000", false},
		{"all same digit", "11111111111111", false},
		{"wrong check digit", "11222333000182", false},
		{"too short", "1122233300018", false},
		{"empty", "", false},
		{"valid CNPJ 2", "45.997.418/0001-53", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validation.ValidateCNPJ(tc.cnpj); got != tc.valid {
				t.Errorf("ValidateCNPJ(%q) = %v, want %v", tc.cnpj, got, tc.valid)
			}
		})
	}
}

func TestValidateCPF(t *testing.T) {
	cases := []struct {
		name  string
		cpf   string
		valid bool
	}{
		{"valid CPF formatted", "529.982.247-25", true},
		{"valid CPF digits", "52998224725", true},
		{"all zeros", "00000000000", false},
		{"all same digit", "11111111111", false},
		{"wrong check digit", "52998224726", false},
		{"too short", "5299822472", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validation.ValidateCPF(tc.cpf); got != tc.valid {
				t.Errorf("ValidateCPF(%q) = %v, want %v", tc.cpf, got, tc.valid)
			}
		})
	}
}

func TestValidateCNPJOrCPF(t *testing.T) {
	if !validation.ValidateCNPJOrCPF("529.982.247-25") {
		t.Error("expected valid CPF")
	}
	if !validation.ValidateCNPJOrCPF("11.222.333/0001-81") {
		t.Error("expected valid CNPJ")
	}
	if validation.ValidateCNPJOrCPF("123") {
		t.Error("expected invalid")
	}
}
