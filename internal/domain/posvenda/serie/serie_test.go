package serie

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestTransicao(t *testing.T) {
	casos := []struct {
		atual, evento, depois string
		erro                  bool
	}{
		{EmEstoque, EventoVenda, NoCliente, false},
		{NoCliente, EventoVenda, "", true},
		{NoCliente, EventoEstornoVenda, EmEstoque, false},
		{EmEstoque, EventoEstornoVenda, "", true},
		{NoCliente, EventoInstalacao, NoCliente, false},
		{EmEstoque, EventoInstalacao, "", true},
		{NoCliente, EventoTransferencia, NoCliente, false},
		{EmAssistencia, EventoTransferencia, "", true},
		{EmEstoque, EventoBaixa, Baixado, false},
		{EmAssistencia, EventoBaixa, Baixado, false},
		{Baixado, EventoBaixa, "", true},
		{Baixado, EventoVenda, "", true},
	}
	for _, c := range casos {
		got, err := Transicao(c.atual, c.evento)
		if (err != nil) != c.erro || got != c.depois {
			t.Errorf("%s + %s: obtido %q, erro %v", c.atual, c.evento, got, err)
		}
	}
}

func TestConferirLista(t *testing.T) {
	got, err := ConferirLista([]string{" A1 ", "", "B2"}, decimal.NewFromInt(2))
	if err != nil || len(got) != 2 || got[0] != "A1" {
		t.Fatalf("lista limpa esperada: %v %v", got, err)
	}
	if _, err := ConferirLista([]string{"A1", "a1"}, decimal.NewFromInt(2)); err == nil {
		t.Error("série repetida (sem diferenciar maiúscula) deve ser recusada")
	}
	if _, err := ConferirLista([]string{"A1"}, decimal.NewFromInt(2)); err == nil {
		t.Error("menos séries que unidades deve ser recusado")
	}
	if _, err := ConferirLista([]string{"A1", "A2"}, decimal.RequireFromString("1.5")); err == nil {
		t.Error("quantidade fracionada deve ser recusada")
	}
	if got, err := ConferirLista(nil, decimal.Zero); err != nil || len(got) != 0 {
		t.Errorf("quantidade zero sem série é válida: %v %v", got, err)
	}
}
