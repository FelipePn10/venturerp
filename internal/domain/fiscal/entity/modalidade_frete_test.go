package entity

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestModalidadeFreteNFe(t *testing.T) {
	s := func(v string) *string { return &v }
	zero, dez := decimal.Zero, decimal.NewFromInt(10)
	casos := []struct {
		tipo  *string
		frete decimal.Decimal
		quer  int
	}{
		{s("CIF"), zero, 0}, {s("FOB"), zero, 1}, {s("FOB"), dez, 1}, {s("TERCEIROS"), zero, 2}, {s("SEM_FRETE"), zero, 9},
		{nil, dez, 0}, {nil, zero, 9}, {s("XYZ"), zero, 9},
	}
	for _, c := range casos {
		if got := ModalidadeFreteNFe(c.tipo, c.frete); got != c.quer {
			t.Errorf("tipo %v frete %s: %d, quer %d", c.tipo, c.frete, got, c.quer)
		}
	}
}

// O pedido herda o tipo comercial do orçamento; a nota precisa de um dos quatro.
func TestTipoFreteDaNota(t *testing.T) {
	casos := map[string]string{
		"CIF CONTRAT.": "CIF", "cif próprio": "CIF", "DAF": "CIF", "CORTESIA": "CIF",
		"FOB CONTRAT.": "FOB", "FOB PROPRIO": "FOB", "RETIRA": "FOB", " fob ": "FOB",
		"TERCEIROS": "TERCEIROS", "SEM FRETE": "SEM_FRETE", "SEM_FRETE": "SEM_FRETE",
		"CONVENIO": "", "": "",
	}
	for entrada, quer := range casos {
		got, ok := TipoFreteDaNota(entrada)
		if got != quer || ok != (quer != "") {
			t.Errorf("%q: %q/%v, quer %q", entrada, got, ok, quer)
		}
	}
}
