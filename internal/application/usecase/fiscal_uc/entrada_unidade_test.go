package fiscal_uc

import (
	"context"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entrada"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type docsUnidade struct {
	repository.FiscalEntryDocumentRepository
	cadastro map[int64]repository.ItemCandidate
	fatores  map[string]decimal.Decimal // "item:DE:PARA"
}

func (d docsUnidade) ItemsByCode(_ context.Context, codes []int64) (map[int64]repository.ItemCandidate, error) {
	out := map[int64]repository.ItemCandidate{}
	for _, c := range codes {
		if it, ok := d.cadastro[c]; ok {
			out[c] = it
		}
	}
	return out, nil
}

func (d docsUnidade) FatorDeConversao(_ context.Context, item int64, de, para string) (*decimal.Decimal, error) {
	if f, ok := d.fatores[strings.Join([]string{decimal.NewFromInt(item).String(), de, para}, ":")]; ok {
		return &f, nil
	}
	return nil, nil
}

func linhaNota(seq int, item int64, uom string, qtd float64) *entity.FiscalEntryItem {
	u := uom
	um := decimal.NewFromInt(1)
	q := decimal.NewFromFloat(qtd)
	return &entity.FiscalEntryItem{Sequence: seq, ItemCode: &item, UOM: &u, Quantity: qtd, MovimentaEstoque: true, FatorConversao: &um, QuantidadeEstoque: &q}
}

// Caso real (Tecnofer, NF 78402): tubo estocado em metros comprado em quilos.
// 115,8975 KG × 0,15530965 M/KG = 18 M — e não 115,8975 "metros".
func TestEntradaConverteParaUnidadeDeEstoque(t *testing.T) {
	ctx := context.Background()
	docs := docsUnidade{
		cadastro: map[int64]repository.ItemCandidate{
			238: {Code: 238, BusinessCode: "601", UOM: "M"},
			239: {Code: 239, BusinessCode: "106", UOM: "M"},
			240: {Code: 240, BusinessCode: "900", UOM: "KG"},
		},
		fatores: map[string]decimal.Decimal{"238:KG:M": decimal.RequireFromString("0.1553096500")},
	}
	s := &EntradaServico{Docs: docs}
	nota := &entity.FiscalEntry{Itens: []*entity.FiscalEntryItem{
		linhaNota(1, 238, "KG", 115.8975),
		linhaNota(2, 239, "KG", 19.4371),
		linhaNota(3, 240, "kg", 10),
	}}

	mudou, err := s.AplicarConversoes(ctx, nota)
	if err != nil || !mudou {
		t.Fatalf("esperava conversão aplicada: mudou=%v err=%v", mudou, err)
	}
	if got := nota.Itens[0].QuantidadeEstoque.StringFixed(4); got != "18.0000" {
		t.Fatalf("601: 115,8975 KG deveria virar 18 M, virou %s", got)
	}
	if !nota.Itens[1].FatorConversao.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("106 não tem conversão: não pode inventar fator")
	}

	pend, err := s.pendenciasDeUnidade(ctx, nota)
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 1 || pend[0].Nivel != entrada.NivelImpede || !strings.Contains(pend[0].Mensagem, "item 2 (106)") ||
		!strings.Contains(pend[0].Mensagem, "KG→M") {
		t.Fatalf("só a linha sem conversão (106) deveria impedir a aprovação: %+v", pend)
	}

	// Fator informado à mão (≠ 1) vale mais que o cadastro.
	manual := decimal.RequireFromString("0.5")
	nota.Itens[1].FatorConversao = &manual
	if pend, _ := s.pendenciasDeUnidade(ctx, nota); len(pend) != 0 {
		t.Fatalf("com fator informado a linha está resolvida: %+v", pend)
	}
	// Linha que não entra no estoque (serviço, uso e consumo) não é barrada.
	nota.Itens[1].FatorConversao = nil
	nota.Itens[1].MovimentaEstoque = false
	if pend, _ := s.pendenciasDeUnidade(ctx, nota); len(pend) != 0 {
		t.Fatalf("linha sem estoque não deveria impedir: %+v", pend)
	}
}
