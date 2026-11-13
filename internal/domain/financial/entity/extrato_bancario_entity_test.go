package entity

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func ptr[T any](v T) *T { return &v }

func linhaExtrato(tipo ExtratoTipo, valor string) ExtratoBancario {
	return ExtratoBancario{ID: 1, ContaBancariaID: 2, Valor: decimal.RequireFromString(valor), Tipo: tipo, Hash: "h1"}
}

func movimento(tipo FluxoCaixaTipo, conta int64, valor string) FluxoCaixa {
	return FluxoCaixa{ID: 9, Tipo: tipo, ContaBancariaID: ptr(conta), Valor: decimal.RequireFromString(valor)}
}

func TestConciliacaoAceitaMesmaContaSentidoEValor(t *testing.T) {
	e := linhaExtrato(ExtratoTipoDebito, "13494.86")
	if m := e.MotivoParaNaoConciliar(movimento(FluxoCaixaTipoSaida, 2, "13494.860")); m != "" {
		t.Fatalf("deveria conciliar, recusou: %s", m)
	}
}

// Cada regra tem o seu caso: uma regra que recusa tudo e uma que aceita tudo
// passariam pela metade destes testes.
func TestConciliacaoRecusaOQueNaoConfere(t *testing.T) {
	casos := []struct {
		nome   string
		linha  ExtratoBancario
		mov    FluxoCaixa
		trecho string
	}{
		{"outra conta", linhaExtrato(ExtratoTipoDebito, "100"), movimento(FluxoCaixaTipoSaida, 3, "100"), "outra conta"},
		{"sem conta", linhaExtrato(ExtratoTipoDebito, "100"), FluxoCaixa{Tipo: FluxoCaixaTipoSaida, Valor: decimal.NewFromInt(100)}, "outra conta"},
		{"débito × entrada", linhaExtrato(ExtratoTipoDebito, "100"), movimento(FluxoCaixaTipoEntrada, 2, "100"), "saída (débito)"},
		{"crédito × saída", linhaExtrato(ExtratoTipoCredito, "100"), movimento(FluxoCaixaTipoSaida, 2, "100"), "entrada (crédito)"},
		{"valor diferente", linhaExtrato(ExtratoTipoCredito, "100"), movimento(FluxoCaixaTipoEntrada, 2, "100.01"), "diferente"},
	}
	for _, c := range casos {
		if m := c.linha.MotivoParaNaoConciliar(c.mov); !strings.Contains(m, c.trecho) {
			t.Errorf("%s: motivo = %q, esperado conter %q", c.nome, m, c.trecho)
		}
	}

	ja := linhaExtrato(ExtratoTipoCredito, "100")
	ja.Conciliado = true
	if m := ja.MotivoParaNaoConciliar(movimento(FluxoCaixaTipoEntrada, 2, "100")); !strings.Contains(m, "já está conciliado") {
		t.Errorf("linha já conciliada: %q", m)
	}
	mov := movimento(FluxoCaixaTipoEntrada, 2, "100")
	mov.Conciliado = true
	if m := linhaExtrato(ExtratoTipoCredito, "100").MotivoParaNaoConciliar(mov); !strings.Contains(m, "outro lançamento") {
		t.Errorf("movimento já conciliado: %q", m)
	}
}

func TestTipoNoCaixaSegueOSentidoDoBanco(t *testing.T) {
	if linhaExtrato(ExtratoTipoDebito, "1").TipoNoCaixa() != FluxoCaixaTipoSaida {
		t.Error("débito deveria ser saída")
	}
	if linhaExtrato(ExtratoTipoCredito, "1").TipoNoCaixa() != FluxoCaixaTipoEntrada {
		t.Error("crédito deveria ser entrada")
	}
}

func TestMovimentoLancadoPeloExtratoSoQuandoNasceuDaLinha(t *testing.T) {
	e := linhaExtrato(ExtratoTipoDebito, "8.50")
	avulso := movimento(FluxoCaixaTipoSaida, 2, "8.50")
	avulso.ExtratoHash = ptr("h1")
	if !e.MovimentoLancadoPeloExtrato(avulso) {
		t.Error("movimento com o hash da linha e sem título deveria ser do extrato")
	}
	// A baixa de um título nunca é desfeita pela conciliação, mesmo que alguém
	// tenha gravado o hash nela.
	baixa := avulso
	baixa.ContasPagarID = ptr(int64(4))
	if e.MovimentoLancadoPeloExtrato(baixa) {
		t.Error("baixa de título não pode ser tratada como lançamento do extrato")
	}
	outro := movimento(FluxoCaixaTipoSaida, 2, "8.50")
	if e.MovimentoLancadoPeloExtrato(outro) {
		t.Error("movimento sem hash não é do extrato")
	}
}

func TestLancarRecusaLinhaConciliada(t *testing.T) {
	e := linhaExtrato(ExtratoTipoDebito, "8.50")
	if m := e.MotivoParaNaoLancar(); m != "" {
		t.Fatalf("deveria lançar: %s", m)
	}
	e.Conciliado = true
	if e.MotivoParaNaoLancar() == "" {
		t.Error("linha conciliada não pode virar outro movimento")
	}
}
