package aprovacao

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func dp(s string) *decimal.Decimal { d := decimal.RequireFromString(s); return &d }
func ip(v int64) *int64            { return &v }

func politicaTeste() Politica {
	return Politica{Tipo: LiberacaoCredito, Ativa: true, Segregacao: true, Niveis: []Nivel{
		{ValorAte: dp("10000"), PerfilID: ip(1)},
		{ValorAte: dp("50000"), PerfilID: ip(2)},
		{PerfilID: ip(3)},
	}}
}

func TestPoliticaValidaENiveisEmEscalada(t *testing.T) {
	p := politicaTeste()
	if err := p.Validar(); err != nil {
		t.Fatal(err)
	}
	for valor, esperado := range map[string]int{"5000": 1, "10000": 1, "10000.01": 2, "50000": 2, "900000": 3} {
		if got := p.NiveisExigidos(decimal.RequireFromString(valor)); got != esperado {
			t.Errorf("valor %s: %d níveis, esperado %d", valor, got, esperado)
		}
	}
	ruins := []Politica{
		{Tipo: "X", Niveis: []Nivel{{PerfilID: ip(1)}}},
		{Tipo: LiberacaoCredito},
		{Tipo: LiberacaoCredito, Niveis: []Nivel{{PerfilID: ip(1)}, {PerfilID: ip(2)}}},
		{Tipo: LiberacaoCredito, Niveis: []Nivel{{ValorAte: dp("100"), PerfilID: ip(1)}, {ValorAte: dp("50"), PerfilID: ip(2)}}},
		{Tipo: LiberacaoCredito, Niveis: []Nivel{{PerfilID: ip(1), UserID: &uuid.Nil}}},
	}
	for i, r := range ruins {
		if r.Validar() == nil {
			t.Errorf("política ruim %d aceita", i)
		}
	}
}

func TestSegregacaoEAprovador(t *testing.T) {
	p := politicaTeste()
	_ = p.Validar()
	pedinte, aprovador1, aprovador2 := uuid.New(), uuid.New(), uuid.New()
	s := Solicitacao{Status: Pendente, NivelAtual: 1, NiveisExigidos: 2, Solicitante: pedinte}
	if ConferirAprovador(p, s, pedinte, true) == nil {
		t.Fatal("quem pediu não aprova")
	}
	if ConferirAprovador(p, s, aprovador1, false) == nil {
		t.Fatal("sem o perfil do nível não aprova")
	}
	if err := ConferirAprovador(p, s, aprovador1, true); err != nil {
		t.Fatal(err)
	}
	s.NivelAtual = 2
	s.Eventos = []Evento{{Evento: "APROVADA_NIVEL", Ator: aprovador1}}
	if ConferirAprovador(p, s, aprovador1, true) == nil {
		t.Fatal("a mesma pessoa não aprova dois níveis")
	}
	if err := ConferirAprovador(p, s, aprovador2, true); err != nil {
		t.Fatal(err)
	}
	p.Segregacao = false
	s.NivelAtual = 1
	s.Eventos = nil
	if err := ConferirAprovador(p, s, pedinte, true); err != nil {
		t.Fatalf("sem segregação quem pediu pode aprovar: %v", err)
	}
	s.Status = Aprovada
	if ConferirAprovador(p, s, aprovador2, true) == nil {
		t.Fatal("solicitação concluída não se aprova de novo")
	}
}
