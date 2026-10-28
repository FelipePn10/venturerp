package entity

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewEntryOperationType_Validation(t *testing.T) {
	if _, err := NewEntryOperationType(0, "x", "1102", uuid.New()); err == nil {
		t.Error("expected error for code 0")
	}
	if _, err := NewEntryOperationType(1, "", "1102", uuid.New()); err == nil {
		t.Error("expected error for empty description")
	}
	if _, err := NewEntryOperationType(1, "x", "", uuid.New()); err == nil {
		t.Error("expected error for empty nature")
	}
	if _, err := NewEntryOperationType(1, "x", "9999", uuid.New()); err == nil {
		t.Error("expected error for nature not starting with 1/2/3")
	}
	o, err := NewEntryOperationType(1, "Compra dentro do estado", "1102", uuid.New())
	if err != nil || !o.IsActive {
		t.Fatalf("expected valid active operation, err=%v", err)
	}
}

func TestEntryOperationType_ValidateUF(t *testing.T) {
	cases := []struct {
		name      string
		nature    string
		ufInGroup bool
		wantErr   bool
	}{
		{"nature1 in group → ok", "1102", true, false},
		{"nature1 NOT in group → erro", "1102", false, true},
		{"nature2 NOT in group → ok", "2102", false, false},
		{"nature2 in group → erro", "2102", true, true},
		{"nature3 foreign → ok regardless", "3102", false, false},
		{"nature3 foreign → ok even if in group", "3102", true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := &EntryOperationType{NatureOperation: tc.nature}
			err := o.ValidateUF("PR", tc.ufInGroup)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateUF(nature=%s, inGroup=%v) err=%v, wantErr=%v", tc.nature, tc.ufInGroup, err, tc.wantErr)
			}
		})
	}
}

func TestNaturezaECreditosDaOperacao(t *testing.T) {
	if n, err := NormalizarNatureza(" 1.101 "); err != nil || n != "1101" {
		t.Fatalf("1.101 → 1101: %q %v", n, err)
	}
	for _, ruim := range []string{"110", "51010", "5101", "abc"} {
		if _, err := NormalizarNatureza(ruim); err == nil {
			t.Errorf("natureza %q deveria ser recusada", ruim)
		}
	}
	casos := []struct {
		natureza  string
		icms, ipi bool
		ok        bool
	}{
		{"1556", true, false, false}, // uso e consumo não credita ICMS
		{"2556", false, true, false}, // nem IPI
		{"1556", false, false, true},
		{"1551", false, true, false}, // ativo não credita IPI
		{"1551", true, false, false}, // ICMS do ativo vai pelo CIAP
		{"1551", false, false, true},
		{"1101", true, true, true},
	}
	for _, c := range casos {
		o := &EntryOperationType{NatureOperation: c.natureza, CreditaICMS: c.icms, CreditaIPI: c.ipi}
		if err := o.ConferirCreditos(); (err == nil) != c.ok {
			t.Errorf("%s icms=%v ipi=%v: err=%v", c.natureza, c.icms, c.ipi, err)
		}
	}
	g := int64(7)
	o := &EntryOperationType{Code: 3, Description: "Compra SP", StateGroupCode: &g}
	if err := o.ValeParaFornecedor("PR", false); err == nil {
		t.Error("fornecedor fora do grupo deveria ser recusado")
	}
	if err := o.ValeParaFornecedor("SP", true); err != nil {
		t.Error(err)
	}
}
