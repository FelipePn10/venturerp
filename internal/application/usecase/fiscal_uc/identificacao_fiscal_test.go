package fiscal_uc

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	fiscalrepository "github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/focusnfe"
)

type fonteDeIdentificacao map[int64]fiscalrepository.IdentificacaoFiscal

func (f fonteDeIdentificacao) IdentificacaoFiscal(context.Context, []int64) (map[int64]fiscalrepository.IdentificacaoFiscal, error) {
	return f, nil
}

func TestAnexarIdentificacaoPoeGTINEUnidadeTributavel(t *testing.T) {
	um, dois, tres := int64(1), int64(2), int64(3)
	itens := []*entity.FiscalExitItem{
		{ItemCode: &um, Quantity: decimal.NewFromInt(10), TotalPrice: decimal.NewFromInt(100)},
		{ItemCode: &dois, Quantity: decimal.NewFromInt(5), TotalPrice: decimal.NewFromInt(50)},
		{ItemCode: &tres, Quantity: decimal.NewFromInt(5), TotalPrice: decimal.NewFromInt(50)},
		{Quantity: decimal.NewFromInt(1), TotalPrice: decimal.NewFromInt(1)},
	}
	payload := focusnfe.NFEPayload{Items: []focusnfe.NFEItem{{UnidadeComercial: "UN"}, {UnidadeComercial: "UN"}, {UnidadeComercial: "CX"}, {UnidadeComercial: "UN"}}}
	fonte := fonteDeIdentificacao{
		1: {GTIN: "7891000100103", UnidadeComercial: "UN", UnidadeTributavel: "KG", FatorTributavel: decimal.RequireFromString("2.5")},
		2: {GTIN: "7891000100103", GTINTributavel: "17891000100100", UnidadeComercial: "UN"},
		3: {UnidadeComercial: "UN", UnidadeTributavel: "KG", FatorTributavel: decimal.NewFromInt(2)},
	}
	if err := anexarIdentificacao(context.Background(), fonte, itens, &payload); err != nil {
		t.Fatal(err)
	}
	a := payload.Items[0]
	if a.CodigoBarrasComercial != "7891000100103" || a.CodigoBarrasTributavel != "7891000100103" {
		t.Fatalf("o cEANTrib acompanha o cEAN: %+v", a)
	}
	if a.UnidadeTributavel != "KG" || a.QuantidadeTributavel != 25 || a.ValorUnitarioTributavel != 4 {
		t.Fatalf("10 UN × 2,5 = 25 KG, R$ 100 / 25 = R$ 4: %+v", a)
	}
	if b := payload.Items[1]; b.CodigoBarrasTributavel != "17891000100100" || b.UnidadeTributavel != "" {
		t.Fatalf("GTIN tributável próprio, sem unidade própria: %+v", b)
	}
	if c := payload.Items[2]; c.UnidadeTributavel != "" {
		t.Fatalf("linha em outra unidade não aplica o fator: %+v", c)
	}
	if d := payload.Items[3]; d.CodigoBarrasComercial != "" {
		t.Fatalf("linha sem item fica como está: %+v", d)
	}
}

func TestDescricaoFixaVenceALinha(t *testing.T) {
	um := int64(1)
	itens := []*entity.FiscalExitItem{{ItemCode: &um, Quantity: decimal.NewFromInt(1), TotalPrice: decimal.NewFromInt(1)}}
	payload := focusnfe.NFEPayload{Items: []focusnfe.NFEItem{{Descricao: "digitada na nota", UnidadeComercial: "UN"}}}
	fonte := fonteDeIdentificacao{1: {UnidadeComercial: "UN", DescricaoFixa: "DO CADASTRO"}}
	if err := anexarIdentificacao(context.Background(), fonte, itens, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Items[0].Descricao != "DO CADASTRO" {
		t.Fatalf("item que não permite alterar sai com a descrição do cadastro: %s", payload.Items[0].Descricao)
	}
}

// Homologação da Tecnofer, 06/11/2026: indEscala sem CEST reprovou no schema e a
// nota nem chegou à SEFAZ.
func TestEscalaEFabricanteSoVaoComCEST(t *testing.T) {
	um, dois, verdadeiro := int64(1), int64(2), true
	itens := []*entity.FiscalExitItem{
		{ItemCode: &um, Quantity: decimal.NewFromInt(1), TotalPrice: decimal.NewFromInt(1)},
		{ItemCode: &dois, Quantity: decimal.NewFromInt(1), TotalPrice: decimal.NewFromInt(1)},
	}
	payload := focusnfe.NFEPayload{Items: []focusnfe.NFEItem{{UnidadeComercial: "UN"}, {UnidadeComercial: "UN"}}}
	fonte := fonteDeIdentificacao{
		1: {UnidadeComercial: "UN", EscalaRelevante: &verdadeiro, CNPJFabricante: "78958717001614"},
		2: {UnidadeComercial: "UN", EscalaRelevante: &verdadeiro, CNPJFabricante: "78958717001614", CEST: "0107500"},
	}
	if err := anexarIdentificacao(context.Background(), fonte, itens, &payload); err != nil {
		t.Fatal(err)
	}
	if a := payload.Items[0]; a.EscalaRelevante != nil || a.CNPJFabricante != "" || a.CEST != "" {
		t.Fatalf("sem CEST o grupo fica de fora: %+v", a)
	}
	if b := payload.Items[1]; b.EscalaRelevante == nil || b.CNPJFabricante == "" || b.CEST != "0107500" {
		t.Fatalf("com CEST o grupo vai inteiro: %+v", b)
	}
}

func TestLinhaSemCodigoEDescricaoUsaOCadastro(t *testing.T) {
	um := int64(5)
	proprio, digitada := "DO-CLIENTE", "DIGITADA"
	itens := []*entity.FiscalExitItem{
		{ItemCode: &um, Quantity: decimal.NewFromInt(1), TotalPrice: decimal.NewFromInt(1)},
		{ItemCode: &um, Quantity: decimal.NewFromInt(1), TotalPrice: decimal.NewFromInt(1), CodigoProduto: &proprio, Description: &digitada},
	}
	payload := focusnfe.NFEPayload{Items: []focusnfe.NFEItem{
		{CodigoProduto: "5", Descricao: "Produto 5", UnidadeComercial: "UN"},
		{CodigoProduto: proprio, Descricao: digitada, UnidadeComercial: "UN"},
	}}
	fonte := fonteDeIdentificacao{5: {UnidadeComercial: "UN", CodigoComercial: "RN-01001", DescricaoCadastro: "BALANCA"}}
	if err := anexarIdentificacao(context.Background(), fonte, itens, &payload); err != nil {
		t.Fatal(err)
	}
	if a := payload.Items[0]; a.CodigoProduto != "RN-01001" || a.Descricao != "BALANCA" {
		t.Fatalf("a chave interna não vai à nota: %+v", a)
	}
	if b := payload.Items[1]; b.CodigoProduto != proprio || b.Descricao != digitada {
		t.Fatalf("o que veio na linha vale: %+v", b)
	}
}

func TestLinhaEmEmbalagemChegaAUnidadeTributavel(t *testing.T) {
	um := int64(1)
	itens := []*entity.FiscalExitItem{{ItemCode: &um, Quantity: decimal.NewFromInt(3), TotalPrice: decimal.NewFromInt(750)}}
	payload := focusnfe.NFEPayload{Items: []focusnfe.NFEItem{{UnidadeComercial: "CX"}}}
	fonte := fonteDeIdentificacao{1: {
		GTIN: "7891000100103", UnidadeComercial: "UN", UnidadeTributavel: "KG", FatorTributavel: decimal.RequireFromString("2.5"),
		GTINPorUnidade: map[string]string{"CX": "17891000100100"}, QuantidadePorUnidade: map[string]decimal.Decimal{"CX": decimal.NewFromInt(10)},
	}}
	if err := anexarIdentificacao(context.Background(), fonte, itens, &payload); err != nil {
		t.Fatal(err)
	}
	a := payload.Items[0]
	if a.CodigoBarrasComercial != "17891000100100" || a.CodigoBarrasTributavel != "7891000100103" {
		t.Fatalf("cEAN da caixa, cEANTrib da unidade: %+v", a)
	}
	if a.UnidadeTributavel != "KG" || a.QuantidadeTributavel != 75 || a.ValorUnitarioTributavel != 10 {
		t.Fatalf("3 CX × 10 UN × 2,5 = 75 KG, R$ 750 / 75 = R$ 10: %+v", a)
	}
}

func TestDescricaoDeExportacaoVenceADoCadastro(t *testing.T) {
	um := int64(1)
	itens := []*entity.FiscalExitItem{{ItemCode: &um, Quantity: decimal.NewFromInt(1), TotalPrice: decimal.NewFromInt(1)}}
	payload := focusnfe.NFEPayload{Items: []focusnfe.NFEItem{{Descricao: "Produto 1", UnidadeComercial: "UN", CFOP: "7101"}}}
	fonte := fonteDeIdentificacao{1: {UnidadeComercial: "UN", DescricaoCadastro: "BALANCA", DescricaoExportacao: "SCALE"}}
	if err := anexarIdentificacao(context.Background(), fonte, itens, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Items[0].Descricao != "SCALE" {
		t.Fatalf("a exportação sai na descrição em inglês: %s", payload.Items[0].Descricao)
	}
}

func TestPaleteSemGTINNaoHerdaOCodigoDaUnidade(t *testing.T) {
	um := int64(1)
	itens := []*entity.FiscalExitItem{{ItemCode: &um, Quantity: decimal.NewFromInt(2), TotalPrice: decimal.NewFromInt(800)}}
	payload := focusnfe.NFEPayload{Items: []focusnfe.NFEItem{{UnidadeComercial: "PL"}}}
	fonte := fonteDeIdentificacao{1: {
		GTIN: "7891000100103", UnidadeComercial: "UN", UnidadeTributavel: "UN", FatorTributavel: decimal.NewFromInt(1),
		GTINPorUnidade: map[string]string{"PL": ""}, QuantidadePorUnidade: map[string]decimal.Decimal{"PL": decimal.NewFromInt(400)},
	}}
	if err := anexarIdentificacao(context.Background(), fonte, itens, &payload); err != nil {
		t.Fatal(err)
	}
	a := payload.Items[0]
	if a.CodigoBarrasComercial != "" || a.CodigoBarrasTributavel != "" {
		t.Fatalf("só o cEANTrib rejeita com 886: o palete sem GTIN leva os dois vazios: %+v", a)
	}
	if a.UnidadeTributavel != "UN" || a.QuantidadeTributavel != 800 || a.ValorUnitarioTributavel != 1 {
		t.Fatalf("2 PL × 400 UN = 800 UN, R$ 800 / 800 = R$ 1: %+v", a)
	}
}

func TestCaixaComGTINEUnidadeSemGTINSaiSemGTIN(t *testing.T) {
	um := int64(1)
	itens := []*entity.FiscalExitItem{{ItemCode: &um, Quantity: decimal.NewFromInt(1), TotalPrice: decimal.NewFromInt(100)}}
	payload := focusnfe.NFEPayload{Items: []focusnfe.NFEItem{{UnidadeComercial: "CX"}}}
	fonte := fonteDeIdentificacao{1: {
		UnidadeComercial: "UN", UnidadeTributavel: "UN", FatorTributavel: decimal.NewFromInt(1),
		GTINPorUnidade: map[string]string{"CX": "17891000100100"}, QuantidadePorUnidade: map[string]decimal.Decimal{"CX": decimal.NewFromInt(10)},
	}}
	if err := anexarIdentificacao(context.Background(), fonte, itens, &payload); err != nil {
		t.Fatal(err)
	}
	if a := payload.Items[0]; a.CodigoBarrasComercial != "" || a.CodigoBarrasTributavel != "" {
		t.Fatalf("cEAN sem cEANTrib é recusado: os dois saem SEM GTIN: %+v", a)
	}
}
