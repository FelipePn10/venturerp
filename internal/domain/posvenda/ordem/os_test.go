package ordem

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestCobertura(t *testing.T) {
	g := Cobertura{EmGarantia: true, CobrePecas: true, CobreMaoObra: true}
	if g.PecaCobrada() || g.HoraCobrada(HoraTrabalho) {
		t.Error("em garantia com peças e mão de obra: peça e trabalho não são cobrados")
	}
	if !g.HoraCobrada(HoraDeslocamento) || !g.DespesaCobrada() {
		t.Error("deslocamento fora da cobertura é cobrado")
	}
	fora := Cobertura{EmGarantia: false, CobrePecas: true, CobreMaoObra: true, CobreDeslocamento: true}
	if !fora.PecaCobrada() || !fora.HoraCobrada(HoraTrabalho) || !fora.DespesaCobrada() {
		t.Error("fora da garantia tudo é cobrado, mesmo com a cobertura marcada")
	}
}

func TestTransicao(t *testing.T) {
	ok := [][2]string{{Aberta, Agendada}, {Agendada, EmExecucao}, {Aberta, EmExecucao}, {EmExecucao, Concluida}, {Concluida, Faturada}, {Concluida, EmExecucao}, {Aberta, Cancelada}}
	for _, c := range ok {
		if err := Transicao(c[0], c[1]); err != nil {
			t.Errorf("%s → %s deveria passar: %v", c[0], c[1], err)
		}
	}
	nao := [][2]string{{Aberta, Concluida}, {Faturada, EmExecucao}, {Cancelada, Aberta}, {Concluida, Cancelada}, {EmExecucao, Faturada}}
	for _, c := range nao {
		if err := Transicao(c[0], c[1]); err == nil {
			t.Errorf("%s → %s não pode", c[0], c[1])
		}
	}
}

func TestSomar(t *testing.T) {
	d := decimal.RequireFromString
	tot := Somar(
		[]Linha{{Custo: d("100"), Preco: d("150"), Cobrado: false}, {Custo: d("40"), Preco: d("60"), Cobrado: true}},
		[]Linha{{Custo: d("200"), Preco: d("320"), Cobrado: false}},
		[]Linha{{Custo: d("30"), Preco: d("30"), Cobrado: true}},
	)
	if !tot.CustoTotal.Equal(d("370")) || !tot.ACobrar.Equal(d("90")) || !tot.CustoGarantia.Equal(d("300")) {
		t.Fatalf("totais: %+v", tot)
	}
}

func TestConclusaoEAgenda(t *testing.T) {
	if Conclusao("", "trocado") == nil || Conclusao("célula queimada", " ") == nil || Conclusao("célula queimada", "trocada") != nil {
		t.Error("conclusão exige defeito e solução")
	}
	agora := time.Now()
	if Agenda(agora.AddDate(0, 0, -3), agora) == nil || Agenda(agora.AddDate(0, 0, 2), agora) != nil {
		t.Error("agenda no passado é recusada")
	}
	if Lancamentos(Concluida) == nil || Lancamentos(EmExecucao) != nil {
		t.Error("lançamento só com a OS ativa")
	}
}
