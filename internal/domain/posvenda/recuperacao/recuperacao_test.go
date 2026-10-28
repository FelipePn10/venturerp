package recuperacao

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestMudancas(t *testing.T) {
	cem := decimal.NewFromInt(100)
	nc, motivo, hoje := "NOTA_CREDITO", "fora do escopo da garantia do fornecedor", time.Now()
	casos := []struct {
		nome  string
		atual string
		m     Mudanca
		ok    bool
	}{
		{"aceita", Reclamada, Mudanca{Para: Aceita}, true},
		{"recusa sem resposta", Reclamada, Mudanca{Para: Recusada}, false},
		{"recusa com resposta", Aceita, Mudanca{Para: Recusada, Resposta: &motivo}, true},
		{"recuperada parcial", Aceita, Mudanca{Para: Recuperada, ValorRecuperado: decimal.NewFromInt(60), Forma: &nc, RecuperadoEm: &hoje}, true},
		{"recuperada acima", Aceita, Mudanca{Para: Recuperada, ValorRecuperado: decimal.NewFromInt(101), Forma: &nc, RecuperadoEm: &hoje}, false},
		{"recuperada sem forma", Reclamada, Mudanca{Para: Recuperada, ValorRecuperado: cem, RecuperadoEm: &hoje}, false},
		{"recusada é final", Recusada, Mudanca{Para: Aceita}, false},
		{"recuperada é final", Recuperada, Mudanca{Para: Recusada, Resposta: &motivo}, false},
	}
	for _, c := range casos {
		if err := c.m.Conferir(c.atual, cem); (err == nil) != c.ok {
			t.Errorf("%s: erro %v", c.nome, err)
		}
	}
}

func TestAbertura(t *testing.T) {
	if (Reclamacao{SupplierCode: 1, Motivo: "x", ValorReclamado: decimal.NewFromInt(1)}).ConferirAbertura() != nil {
		t.Fatal("válida")
	}
	if (Reclamacao{SupplierCode: 1, Motivo: " ", ValorReclamado: decimal.NewFromInt(1)}).ConferirAbertura() == nil {
		t.Fatal("sem motivo")
	}
	if (Reclamacao{SupplierCode: 1, Motivo: "x"}).ConferirAbertura() == nil {
		t.Fatal("sem valor")
	}
}
