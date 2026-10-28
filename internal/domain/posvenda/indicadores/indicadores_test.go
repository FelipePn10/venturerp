package indicadores

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/posvenda/sla"
)

func TestTaxaEMTBF(t *testing.T) {
	if p := TaxaFalha(3, 40); p == nil || !p.Equal(decimal.RequireFromString("7.5")) {
		t.Fatalf("3 de 40 = 7,5%%: %v", p)
	}
	if TaxaFalha(1, 0) != nil {
		t.Fatal("sem base em campo não há taxa")
	}
	if m := MTBF(decimal.NewFromInt(1000), 3); m == nil || !m.Equal(decimal.RequireFromString("333.3")) {
		t.Fatalf("1000 dias / 3 falhas: %v", m)
	}
	if MTBF(decimal.NewFromInt(1000), 0) != nil {
		t.Fatal("sem falha não há MTBF")
	}
	s := SLA{RespostaAvaliada: 4, RespostaCumprida: 3, SolucaoAvaliada: 0}
	s.Calcular()
	if s.RespostaPct == nil || !s.RespostaPct.Equal(decimal.NewFromInt(75)) || s.SolucaoPct != nil {
		t.Fatalf("SLA: %+v", s)
	}
}

func TestAvaliarSLA(t *testing.T) {
	agora := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	h := func(x int) *time.Time { v := agora.Add(time.Duration(x) * time.Hour); return &v }
	cs := []sla.Chamado{
		{AbertoEm: *h(-10), RespostaAte: h(-8), RespondidoEm: h(-9), SolucaoAte: h(-2), SolucionadoEm: h(-3)}, // cumpriu os dois
		{AbertoEm: *h(-10), RespostaAte: h(-8), RespondidoEm: h(-7), SolucaoAte: h(5)},                        // resposta estourada, solução correndo
		{AbertoEm: *h(-10), RespostaAte: h(-8), SolucaoAte: h(-1)},                                            // vencidos
		{AbertoEm: *h(-10), RespostaAte: h(-8), Cancelado: true},
	}
	var s SLA
	s.Avaliar(cs, agora)
	if s.ComPrazo != 3 || s.RespostaAvaliada != 3 || s.RespostaCumprida != 1 || s.SolucaoAvaliada != 2 || s.SolucaoCumprida != 1 {
		t.Fatalf("%+v", s)
	}
	if !s.SolucaoPct.Equal(decimal.NewFromInt(50)) {
		t.Fatalf("solução 1 de 2: %v", s.SolucaoPct)
	}
}
