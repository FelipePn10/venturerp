package entity

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuildMask(t *testing.T) {
	// out-of-order positions must be sorted; empty values skipped; '#'-joined.
	mask, hash := BuildMask([]MaskSegment{
		{Position: 20, Value: "50"},
		{Position: 10, Value: "VERDE"},
		{Position: 30, Value: ""},
	})
	if mask != "VERDE#50" {
		t.Fatalf("mask = %q, want VERDE#50", mask)
	}
	if len(hash) != 8 {
		t.Fatalf("hash len = %d, want 8", len(hash))
	}
	// deterministic
	if _, h2 := BuildMask([]MaskSegment{{Position: 10, Value: "VERDE"}, {Position: 20, Value: "50"}}); h2 != hash {
		t.Fatalf("hash not deterministic: %s != %s", h2, hash)
	}
}

func TestCharacteristic_Validate(t *testing.T) {
	uid := uuid.New()

	// ESCOLHA without a set is invalid.
	c, _ := NewCharacteristic("COR", "Cor", TypeEscolha, uid)
	if err := c.Validate(); err == nil {
		t.Error("ESCOLHA sem conjunto deveria falhar")
	}
	set := int64(5)
	c.SetID = &set
	if err := c.Validate(); err != nil {
		t.Errorf("ESCOLHA com conjunto deveria passar: %v", err)
	}

	// OPCAO defaults SIM/NAO labels.
	o, _ := NewCharacteristic("OPT", "Opção", TypeOpcao, uid)
	if err := o.Validate(); err != nil {
		t.Fatalf("OPCAO validate: %v", err)
	}
	if o.OptionTrue != "SIM" || o.OptionFalse != "NAO" {
		t.Errorf("OPCAO defaults = %q/%q, want SIM/NAO", o.OptionTrue, o.OptionFalse)
	}

	// INF_NUMERICA with max < min invalid.
	n, _ := NewCharacteristic("LARG", "Largura", TypeInfNumerica, uid)
	lo, hi := 10.0, 5.0
	n.NumMin, n.NumMax = &lo, &hi
	if err := n.Validate(); err == nil {
		t.Error("num_max < num_min deveria falhar")
	}

	// CAMPO requires a field source.
	f, _ := NewCharacteristic("SEQ", "Sequência", TypeCampo, uid)
	if err := f.Validate(); err == nil {
		t.Error("CAMPO sem field_source deveria falhar")
	}
	f.FieldSource = FieldItemCode
	if err := f.Validate(); err != nil {
		t.Errorf("CAMPO com field_source deveria passar: %v", err)
	}

	// invalid type
	bad := &Characteristic{Code: "X", Description: "x", Type: "XPTO"}
	if err := bad.Validate(); err == nil {
		t.Error("tipo inválido deveria falhar")
	}
}

func TestMatchOperator(t *testing.T) {
	cases := []struct {
		op, actual, expected string
		want                 bool
	}{
		{OpEqual, "AZ", "az", true},
		{OpEqual, "AZ", "VE", false},
		{OpDifferent, "AZ", "VE", true},
		{OpDifferent, "AZ", "AZ", false},
		{OpBelongs, "AZ", "AZ,VE,PT", true},
		{OpBelongs, "XX", "AZ,VE", false},
		{OpNotBelongs, "XX", "AZ,VE", true},
		{OpGreater, "B", "A", true},
		{OpLess, "A", "B", true},
	}
	for _, c := range cases {
		if got := MatchOperator(c.op, c.actual, c.expected); got != c.want {
			t.Errorf("MatchOperator(%s,%q,%q) = %v, want %v", c.op, c.actual, c.expected, got, c.want)
		}
	}
}

func TestNewVariable_DefaultsMaskComposition(t *testing.T) {
	v, err := NewVariable(1, "AZ", "Azul", "", uuid.New())
	if err != nil {
		t.Fatalf("NewVariable: %v", err)
	}
	if v.MaskComposition != "AZ" {
		t.Errorf("mask_composition default = %q, want AZ (o código)", v.MaskComposition)
	}
}
