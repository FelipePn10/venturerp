package machine_uc

import (
	"strings"
	"testing"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
)

// O enum de capacidade é validado antes do banco: um valor desconhecido vira
// 422 em PT-BR em vez de um erro cru do Postgres.
func TestNormalizeCapacityUnit(t *testing.T) {
	if got, err := normalizeCapacityUnit("peças"); err != nil || got != types.Pieces {
		t.Fatalf("unidade em minúsculas = %q err=%v", got, err)
	}
	if got, err := normalizeCapacityUnit("  KG "); err != nil || got != types.Kilogram {
		t.Fatalf("unidade com espaços = %q err=%v", got, err)
	}
	_, err := normalizeCapacityUnit("PIECES")
	v, ok := errorsuc.AsValidation(err)
	if !ok {
		t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
	}
	if !strings.Contains(v.Error(), "PEÇAS") {
		t.Fatalf("mensagem não lista as unidades aceitas: %q", v.Error())
	}
	if _, err := normalizeCapacityUnit(""); err == nil {
		t.Fatal("unidade vazia aceita")
	}
}

func TestNormalizeCapacityPeriod(t *testing.T) {
	if got, err := normalizeCapacityPeriod("hora"); err != nil || got != types.Hour {
		t.Fatalf("período em minúsculas = %q err=%v", got, err)
	}
	_, err := normalizeCapacityPeriod("WEEK")
	if _, ok := errorsuc.AsValidation(err); !ok {
		t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
	}
}

// A tela usa fração (0,9) mas o usuário costuma digitar o percentual (90).
func TestNormalizeEfficiency(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
		ok   bool
	}{
		{0.9, 0.9, true},
		{1, 1, true},
		{90, 0.9, true},
		{100, 1, true},
		{0, 0, false},
		{-1, 0, false},
		{101, 0, false},
	}
	for _, tc := range cases {
		got, err := normalizeEfficiency(tc.in)
		if tc.ok != (err == nil) {
			t.Fatalf("eficiência %v: err=%v", tc.in, err)
		}
		if tc.ok && got != tc.want {
			t.Fatalf("eficiência %v = %v, quer %v", tc.in, got, tc.want)
		}
	}
}

func TestValidateMachineFields_NamesTheMissingField(t *testing.T) {
	cases := []struct {
		name     string
		code     int64
		machine  string
		typeCode int64
		capacity float64
		expect   string
	}{
		{"sem código", 0, "Guilhotina", 1, 10, "código da máquina"},
		{"sem nome", 1, "  ", 1, 10, "nome da máquina"},
		{"sem tipo", 1, "Guilhotina", 0, 10, "tipo da máquina"},
		{"sem capacidade", 1, "Guilhotina", 1, 0, "capacidade da máquina"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateMachineFields(tc.code, tc.machine, tc.typeCode, tc.capacity)
			v, ok := errorsuc.AsValidation(err)
			if !ok {
				t.Fatalf("esperado ValidationError, veio %T (%v)", err, err)
			}
			if !strings.Contains(v.Error(), tc.expect) {
				t.Fatalf("mensagem = %q, esperava citar %q", v.Error(), tc.expect)
			}
		})
	}
	if err := validateMachineFields(1, "Guilhotina", 1, 10); err != nil {
		t.Fatalf("máquina válida rejeitada: %v", err)
	}
}

func TestMachineTypeEnum_InvalidValueIsPortuguese(t *testing.T) {
	var parsed types.MachineTypeEnum
	err := parsed.UnmarshalJSON([]byte(`"SERRA"`))
	if err == nil {
		t.Fatal("classificação inexistente aceita")
	}
	if !strings.Contains(err.Error(), "inválida") || !strings.Contains(err.Error(), "CUT") {
		t.Fatalf("mensagem = %q", err.Error())
	}
	if err := parsed.UnmarshalJSON([]byte(`"cut"`)); err != nil || parsed != types.MachineCut {
		t.Fatalf("classificação em minúsculas rejeitada: %v (%q)", err, parsed)
	}
}
