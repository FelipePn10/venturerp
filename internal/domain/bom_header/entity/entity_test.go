package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestCicloDeVidaDaVersao(t *testing.T) {
	casos := []struct {
		de, para string
		ok       bool
	}{
		{StatusDraft, StatusApproved, true},
		{StatusDraft, StatusObsolete, true},
		{StatusApproved, StatusObsolete, true},
		{StatusApproved, StatusDraft, false},
		{StatusObsolete, StatusApproved, false},
		{StatusObsolete, StatusDraft, false},
		{StatusApproved, StatusApproved, false},
	}
	for _, c := range casos {
		err := (&BomHeader{Status: c.de}).PodeIrPara(c.para)
		if (err == nil) != c.ok {
			t.Errorf("%s → %s: ok esperado %v, veio %v", c.de, c.para, c.ok, err)
		}
	}
}

func TestCompararLinhas(t *testing.T) {
	d := decimal.RequireFromString
	aprovada := []Line{
		{ChildCode: 10, Sequence: 1, Quantity: d("2"), Unit: "UN"},
		{ChildCode: 20, Sequence: 2, Quantity: d("1.5"), Unit: "KG"},
		{ChildCode: 30, Sequence: 3, Quantity: d("1"), Unit: "UN"},
	}
	atual := []Line{
		{ChildCode: 10, Sequence: 1, Quantity: d("2.0"), Unit: "UN"}, // igual (2 == 2.0)
		{ChildCode: 20, Sequence: 2, Quantity: d("1.8"), Unit: "KG"}, // alterado
		{ChildCode: 40, Sequence: 4, Quantity: d("3"), Unit: "UN"},   // incluído
	}
	got := CompararLinhas(aprovada, atual)
	if len(got) != 3 {
		t.Fatalf("esperava 3 diferenças, veio %d: %+v", len(got), got)
	}
	tipos := map[int64]string{}
	for _, c := range got {
		tipos[c.ChildCode] = c.Kind
	}
	if tipos[20] != "ALTERADO" || tipos[30] != "REMOVIDO" || tipos[40] != "INCLUIDO" {
		t.Fatalf("classificação errada: %+v", got)
	}
	if len(CompararLinhas(aprovada, aprovada)) != 0 {
		t.Fatal("estrutura igual à aprovada não pode ter diferença")
	}
}
