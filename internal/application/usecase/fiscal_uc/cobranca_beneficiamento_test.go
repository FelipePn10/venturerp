package fiscal_uc

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// A NF-e 5.911 da Usimac: serviço 230,00 (5124) + material do cliente voltando
// 182,60 (5902) = 412,60. A fatura impressa no DANFE é de 230,00. Parcelar sobre
// o total gerava duplicata e título cobrando do cliente o próprio material.
func notaDeBeneficiamento() (*fiscalentity.FiscalExit, []*fiscalentity.FiscalExitItem) {
	nota := &fiscalentity.FiscalExit{ID: 1, NumeroNF: 5911, Serie: "1", ValorTotal: decimal.NewFromFloat(412.60),
		DataEmissao: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)}
	itens := []*fiscalentity.FiscalExitItem{
		{Sequence: 1, Cfop: "5124", Quantity: decimal.NewFromFloat(1), UnitPrice: decimal.NewFromFloat(230), TotalPrice: decimal.NewFromFloat(230)},
		{Sequence: 2, Cfop: "5902", Quantity: decimal.NewFromFloat(3.848), UnitPrice: decimal.NewFromFloat(36.93), TotalPrice: decimal.NewFromFloat(142.11)},
		{Sequence: 3, Cfop: "5902", Quantity: decimal.NewFromFloat(1.304), UnitPrice: decimal.NewFromFloat(31.05), TotalPrice: decimal.NewFromFloat(40.49)},
	}
	return nota, itens
}

func TestPlanoDaNotaDeBeneficiamentoCobraSoOServico(t *testing.T) {
	nota, itens := notaDeBeneficiamento()
	plano, err := resolverPlanoDaNota(context.Background(), nota, itens, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plano.SemCobranca || len(plano.Parcelas) != 1 || !plano.Parcelas[0].Valor.Equal(decimal.RequireFromString("230")) {
		t.Fatalf("a nota cobra só o serviço (230,00), veio %+v", plano)
	}
	payload, _ := montarPayloadNFe(nota, itens, configBase(), plano, TributacaoReforma{})
	if len(payload.FormaPagamento) != 1 || payload.FormaPagamento[0].Valor != 230 {
		t.Fatalf("pagamento informado à SEFAZ deve ser 230,00: %+v", payload.FormaPagamento)
	}
}

func TestNotaSoDeRetornoNaoCobra(t *testing.T) {
	nota := &fiscalentity.FiscalExit{ID: 2, NumeroNF: 10, Serie: "1", ValorTotal: decimal.NewFromFloat(182.60), DataEmissao: time.Now()}
	itens := []*fiscalentity.FiscalExitItem{{Sequence: 1, Cfop: "5902", Quantity: decimal.NewFromFloat(1), UnitPrice: decimal.NewFromFloat(182.60), TotalPrice: decimal.NewFromFloat(182.60)}}
	plano, err := resolverPlanoDaNota(context.Background(), nota, itens, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !plano.SemCobranca || len(plano.Parcelas) != 0 {
		t.Fatalf("retorno puro não tem parcela: %+v", plano)
	}
	payload, _ := montarPayloadNFe(nota, itens, configBase(), plano, TributacaoReforma{})
	if len(payload.FormaPagamento) != 1 || payload.FormaPagamento[0].FormaPagamento != "90" || len(payload.Duplicatas) != 0 {
		t.Fatalf("retorno puro sai \"sem pagamento\" (tPag 90): %+v / %+v", payload.FormaPagamento, payload.Duplicatas)
	}
	if err := gerarTitulos(context.Background(), nil, nota, plano, nil, [16]byte{}); err != nil {
		t.Fatalf("sem cobrança não gera título (nem toca o financeiro): %v", err)
	}
}

func TestVendaComumContinuaCobrandoOTotal(t *testing.T) {
	nota := &fiscalentity.FiscalExit{ID: 3, NumeroNF: 11, Serie: "1", ValorTotal: decimal.NewFromFloat(1000), DataEmissao: time.Now()}
	itens := []*fiscalentity.FiscalExitItem{{Sequence: 1, Cfop: "5101", Quantity: decimal.NewFromFloat(1), UnitPrice: decimal.NewFromFloat(1000), TotalPrice: decimal.NewFromFloat(1000)}}
	plano, err := resolverPlanoDaNota(context.Background(), nota, itens, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plano.Parcelas) != 1 || !plano.Parcelas[0].Valor.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("venda comum: parcela = total, veio %+v", plano.Parcelas)
	}
}
