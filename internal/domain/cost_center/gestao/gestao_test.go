package gestao

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestMontarArvoreConsolida(t *testing.T) {
	p := func(c int32) *int32 { return &c }
	centros := []Centro{
		{ID: 1, Code: 1, Codigo: "CC-FAB"},
		{ID: 2, Code: 2, Codigo: "CC-FAB-CORTE", ParentCode: p(1)},
		{ID: 3, Code: 3, Codigo: "CC-FAB-SOLDA", ParentCode: p(1)},
		{ID: 4, Code: 4, Codigo: "CC-ADM"},
		{ID: 5, Code: 5, Codigo: "ORFAO", ParentCode: p(99)},
	}
	d := decimal.RequireFromString
	orc := map[int64]decimal.Decimal{2: d("1000"), 3: d("500"), 4: d("300")}
	real := map[int64]decimal.Decimal{1: d("10"), 2: d("1200"), 3: d("100"), 4: d("200")}
	raizes := MontarArvore(centros, orc, real)
	if len(raizes) != 3 || raizes[0].Codigo != "CC-ADM" || raizes[2].Codigo != "ORFAO" {
		t.Fatalf("raízes: %v", raizes)
	}
	fab := raizes[1]
	if !fab.OrcadoConsolidado.Equal(d("1500")) || !fab.RealizadoConsolidado.Equal(d("1310")) || fab.Estourado() {
		t.Fatalf("fábrica consolidada: %s/%s", fab.OrcadoConsolidado, fab.RealizadoConsolidado)
	}
	if !fab.Filhos[0].Estourado() || fab.Filhos[1].Estourado() {
		t.Fatal("corte estourou (1200 > 1000); solda não")
	}
	if raizes[0].Estourado() {
		t.Fatal("ADM dentro do orçado")
	}
}

func TestDecidirAprovacaoDaRequisicao(t *testing.T) {
	aprovador, pedinte, outro := uuid.New(), uuid.New(), uuid.New()
	nova := func() *AprovacaoRequisicao {
		return &AprovacaoRequisicao{Status: AprovacaoPendente, AprovadorID: aprovador, Solicitante: &pedinte, CentroCodigo: "CC-FAB"}
	}
	if err := nova().Decidir(outro, true, ""); err == nil {
		t.Fatal("só o aprovador do centro decide")
	}
	if err := nova().Decidir(aprovador, false, " "); err == nil {
		t.Fatal("reprovar exige motivo")
	}
	a := nova()
	if err := a.Decidir(aprovador, true, ""); err != nil || a.Status != AprovacaoAprovada || a.DecididoPor == nil {
		t.Fatalf("aprovação: %v %+v", err, a)
	}
	if err := a.Decidir(aprovador, true, ""); err == nil {
		t.Fatal("decidida não se decide de novo")
	}
	propria := &AprovacaoRequisicao{Status: AprovacaoPendente, AprovadorID: pedinte, Solicitante: &pedinte}
	if err := propria.Decidir(pedinte, true, ""); err == nil {
		t.Fatal("segregação: quem pediu não aprova")
	}
}

func TestCentroRecebeLancamento(t *testing.T) {
	ini := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fim := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	c := Centro{Codigo: "CC-X", IsActive: true, StartDate: ini, EndDate: &fim}
	if err := c.RecebeLancamentoEm(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	if c.RecebeLancamentoEm(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)) == nil {
		t.Fatal("fora da vigência")
	}
	c.IsActive = false
	if c.RecebeLancamentoEm(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)) == nil {
		t.Fatal("inativo")
	}
	g := Gestao{Codigo: " ", AprovaRequisicoes: true}
	if g.Conferir() == nil {
		t.Fatal("código obrigatório")
	}
	g.Codigo = "CC-1"
	if g.Conferir() == nil {
		t.Fatal("aprova requisições sem aprovador")
	}
}
