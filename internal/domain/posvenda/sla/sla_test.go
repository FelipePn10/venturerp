package sla

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestPrazosEmHorasCorridasComFracao(t *testing.T) {
	r := Regra{Prioridade: "HIGH", HorasResposta: decimal.RequireFromString("1.5"), HorasSolucao: decimal.NewFromInt(24), AlertaPct: 80}
	ab := time.Date(2026, 10, 23, 17, 0, 0, 0, time.UTC) // sexta 17h
	p := r.Prazos(ab, nil)
	if !p.RespostaAte.Equal(ab.Add(90*time.Minute)) || !p.SolucaoAte.Equal(time.Date(2026, 10, 24, 17, 0, 0, 0, time.UTC)) {
		t.Fatalf("resposta %v solução %v", p.RespostaAte, p.SolucaoAte)
	}
	// Risco a 80%: 72 min e 19,2 h depois da abertura.
	if !p.RespostaRisco.Equal(ab.Add(72*time.Minute)) || !p.SolucaoRisco.Equal(ab.Add(1152*time.Minute)) {
		t.Fatalf("risco %v / %v", p.RespostaRisco, p.SolucaoRisco)
	}
}

func TestConferir(t *testing.T) {
	ok := Regra{Prioridade: "NORMAL", HorasResposta: decimal.NewFromInt(4), HorasSolucao: decimal.NewFromInt(48), AlertaPct: 80}
	if err := ok.Conferir(); err != nil {
		t.Fatal(err)
	}
	for nome, r := range map[string]Regra{
		"prioridade":         {Prioridade: "URGENTE", HorasResposta: decimal.NewFromInt(1), HorasSolucao: decimal.NewFromInt(2), AlertaPct: 80},
		"solução < resposta": {Prioridade: "LOW", HorasResposta: decimal.NewFromInt(8), HorasSolucao: decimal.NewFromInt(4), AlertaPct: 80},
		"sem horas":          {Prioridade: "LOW", AlertaPct: 80},
		"alerta 100":         {Prioridade: "LOW", HorasResposta: decimal.NewFromInt(1), HorasSolucao: decimal.NewFromInt(2), AlertaPct: 100},
	} {
		if r.Conferir() == nil {
			t.Errorf("%s: devia recusar", nome)
		}
	}
}

func TestEscolherClienteVenceGeralEInativaNaoVale(t *testing.T) {
	c := int64(7)
	regras := []Regra{
		{ID: 1, Prioridade: "HIGH", Ativo: true},
		{ID: 2, Prioridade: "HIGH", CustomerCode: &c, Ativo: true},
		{ID: 3, Prioridade: "LOW", CustomerCode: &c, Ativo: false},
	}
	if r := Escolher(regras, "HIGH", 7); r == nil || r.ID != 2 {
		t.Fatalf("cliente com acordo: %+v", r)
	}
	if r := Escolher(regras, "HIGH", 8); r == nil || r.ID != 1 {
		t.Fatalf("outro cliente usa a geral: %+v", r)
	}
	if r := Escolher(regras, "LOW", 7); r != nil {
		t.Fatalf("regra inativa não vale: %+v", r)
	}
}

func TestSituacao(t *testing.T) {
	ini := time.Date(2026, 10, 25, 8, 0, 0, 0, time.UTC)
	ate := ini.Add(10 * time.Hour)
	at := func(h float64) time.Time { return ini.Add(time.Duration(h * float64(time.Hour))) }
	casos := []struct {
		agora     time.Time
		concluido *time.Time
		quer      string
	}{
		{at(1), nil, NoPrazo},
		{at(7.9), nil, NoPrazo},
		{at(8), nil, EmRisco},
		{at(10), nil, Vencido},
		{at(12), ptr(at(9)), Cumprido},
		{at(12), ptr(at(11)), Estourado},
	}
	for _, c := range casos {
		if s := Situacao(ini, &ate, nil, c.concluido, nil, 80, c.agora); s != c.quer {
			t.Errorf("agora %v: %s, queria %s", c.agora, s, c.quer)
		}
	}
	if s := Situacao(ini, nil, nil, nil, nil, 80, at(100)); s != SemPrazo {
		t.Errorf("sem regra: %s", s)
	}
	// Risco gravado vence o percentual; relógio parado não vence nem arrisca.
	risco := at(5)
	if s := Situacao(ini, &ate, &risco, nil, nil, 80, at(6)); s != EmRisco {
		t.Errorf("risco gravado: %s", s)
	}
	if s := Situacao(ini, &ate, &risco, nil, ptr(at(4)), 80, at(30)); s != Pausado {
		t.Errorf("pausado: %s", s)
	}
	if s := Situacao(ini, &ate, &risco, ptr(at(9)), ptr(at(4)), 80, at(30)); s != Cumprido {
		t.Errorf("concluído conta mesmo pausado: %s", s)
	}
}

func ptr(t time.Time) *time.Time { return &t }
