package fiscal_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entrada"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
	convservice "github.com/FelipePn10/panossoerp/internal/domain/item_conversion/service"
	uomrepo "github.com/FelipePn10/panossoerp/internal/domain/unit_of_measure/repository"
)

// docsUnidade: o cadastro de itens, as conversões (arestas do grafo, como o
// banco devolve), os sinônimos de unidade e o custo da última entrada.
type docsUnidade struct {
	repository.FiscalEntryDocumentRepository
	cadastro map[int64]repository.ItemCandidate
	arestas  map[int64][]convservice.Aresta
	padrao   []convservice.Aresta
	sinonimo map[string]string // texto da nota → unidade do catálogo
	custo    map[int64]decimal.Decimal
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

func (d docsUnidade) ConversaoDeUnidade(_ context.Context, item int64, _ string, _ time.Time, de, para string) (*convservice.Caminho, error) {
	arestas := append(append([]convservice.Aresta{}, d.arestas[item]...), d.padrao...)
	if c, ok := convservice.Resolver(arestas, de, para); ok {
		return &c, nil
	}
	return nil, nil
}

func (d docsUnidade) NormalizarUnidades(_ context.Context, _ *int64, textos []string) (map[string]uomrepo.Resolucao, error) {
	out := map[string]uomrepo.Resolucao{}
	for _, t := range textos {
		r := uomrepo.Resolucao{Texto: t}
		if u, ok := d.sinonimo[strings.ToUpper(t)]; ok {
			r.Unidade, r.Via = u, "SINONIMO"
		}
		out[t] = r
	}
	return out, nil
}

func (d docsUnidade) CustoDeReferencia(_ context.Context, item int64, _ int64) (*decimal.Decimal, error) {
	if c, ok := d.custo[item]; ok {
		return &c, nil
	}
	return nil, nil
}

func aresta(de, para, qtdDe, qtdPara string) convservice.Aresta {
	return convservice.Aresta{De: de, Para: para, QtdDe: decimal.RequireFromString(qtdDe), QtdPara: decimal.RequireFromString(qtdPara), Origem: convservice.OrigemItem}
}

func linhaNota(seq int, item int64, uom string, qtd float64) *entity.FiscalEntryItem {
	u := uom
	um := decimal.NewFromInt(1)
	q := decimal.NewFromFloat(qtd)
	return &entity.FiscalEntryItem{Sequence: seq, ItemCode: &item, UOM: &u, Quantity: decimal.NewFromFloat(qtd), MovimentaEstoque: true, FatorConversao: &um, QuantidadeEstoque: &q}
}

func origem(it *entity.FiscalEntryItem) string { return textoUnidade(it.OrigemConversao) }

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
		arestas: map[int64][]convservice.Aresta{238: {aresta("KG", "M", "1", "0.15530965")}},
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
	if origem(nota.Itens[0]) != OrigemCadastroItem || nota.Itens[0].CaminhoConversao == nil {
		t.Fatalf("601: origem/caminho não registrados: %v", origem(nota.Itens[0]))
	}
	if !nota.Itens[1].FatorConversao.Equal(decimal.NewFromInt(1)) || nota.Itens[1].OrigemConversao != nil {
		t.Fatalf("106 não tem conversão: não pode inventar fator")
	}
	if origem(nota.Itens[2]) != OrigemMesmaUnidade {
		t.Fatalf("900: kg = KG é a mesma unidade, origem %q", origem(nota.Itens[2]))
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

// Chapa comprada em KG, estocada em UN, sem KG→UN direto: KG→M2 do item e
// M2→UN do item encadeiam. "UND" da nota é sinônimo de UN.
func TestEntradaSinonimoEEncadeamento(t *testing.T) {
	ctx := context.Background()
	docs := docsUnidade{
		cadastro: map[int64]repository.ItemCandidate{
			10: {Code: 10, BusinessCode: "CH-3", UOM: "UN"},
			11: {Code: 11, BusinessCode: "PARAF", UOM: "UN"},
		},
		// 1 chapa (UN) = 3 M2; 1 M2 = 23,55 KG → 1 UN = 70,65 KG.
		arestas:  map[int64][]convservice.Aresta{10: {aresta("UN", "M2", "1", "3"), aresta("M2", "KG", "1", "23.55")}},
		sinonimo: map[string]string{"UND": "UN", "PÇ": "UN"},
	}
	s := &EntradaServico{Docs: docs}
	nota := &entity.FiscalEntry{Itens: []*entity.FiscalEntryItem{
		linhaNota(1, 10, "KG", 706.5),
		linhaNota(2, 11, "UND", 500),
	}}
	if _, err := s.AplicarConversoes(ctx, nota); err != nil {
		t.Fatal(err)
	}
	if got := nota.Itens[0].QuantidadeEstoque.StringFixed(4); got != "10.0000" {
		t.Fatalf("706,5 KG deveriam ser 10 chapas, deu %s", got)
	}
	if c := textoUnidade(nota.Itens[0].CaminhoConversao); !strings.Contains(c, "M2") {
		t.Fatalf("o caminho deveria passar por M2: %q", c)
	}
	if origem(nota.Itens[1]) != OrigemSinonimo || textoUnidade(nota.Itens[1].UnidadeNormalizada) != "UN" {
		t.Fatalf("UND deveria ser sinônimo de UN: origem %q, normalizada %q", origem(nota.Itens[1]), textoUnidade(nota.Itens[1].UnidadeNormalizada))
	}
	if pend, _ := s.pendenciasDeUnidade(ctx, nota); len(pend) != 0 {
		t.Fatalf("nada deveria impedir: %+v", pend)
	}
}

// Sem cadastro, o XML declara a unidade tributável: 2 CX = 240 UN.
func TestEntradaUnidadeTributavelDoXML(t *testing.T) {
	ctx := context.Background()
	docs := docsUnidade{cadastro: map[int64]repository.ItemCandidate{
		20: {Code: 20, BusinessCode: "LUVA", UOM: "UN"},
		21: {Code: 21, BusinessCode: "ARAME", UOM: "M"},
	},
		arestas: map[int64][]convservice.Aresta{21: {aresta("KG", "M", "1", "40")}},
	}
	s := &EntradaServico{Docs: docs}
	l1 := linhaNota(1, 20, "CX", 2)
	l1.UnidadeTributavel, l1.QuantidadeTributavel = strPtr("UN"), decPtr("240")
	// 3 RL = 15 KG no XML; o item converte KG→M (1 KG = 40 M): 1 RL = 200 M.
	l2 := linhaNota(2, 21, "RL", 3)
	l2.UnidadeTributavel, l2.QuantidadeTributavel = strPtr("KG"), decPtr("15")
	nota := &entity.FiscalEntry{Itens: []*entity.FiscalEntryItem{l1, l2}}
	if _, err := s.AplicarConversoes(ctx, nota); err != nil {
		t.Fatal(err)
	}
	if origem(l1) != OrigemXMLTributavel || l1.QuantidadeEstoque.StringFixed(2) != "240.00" {
		t.Fatalf("2 CX deveriam entrar como 240 UN pelo XML: %s (%s)", l1.QuantidadeEstoque, origem(l1))
	}
	if origem(l2) != OrigemXMLTributavel || l2.QuantidadeEstoque.StringFixed(2) != "600.00" {
		t.Fatalf("3 RL deveriam entrar como 600 M (XML + cadastro): %s (%s)", l2.QuantidadeEstoque, origem(l2))
	}
}

// Fator decidido (manual, vínculo) não é recalculado pelo cadastro.
func TestEntradaFatorDecididoPrevalece(t *testing.T) {
	ctx := context.Background()
	docs := docsUnidade{
		cadastro: map[int64]repository.ItemCandidate{30: {Code: 30, BusinessCode: "TB", UOM: "M"}},
		arestas:  map[int64][]convservice.Aresta{30: {aresta("KG", "M", "1", "0.5")}},
	}
	s := &EntradaServico{Docs: docs}
	l := linhaNota(1, 30, "KG", 10)
	f := decimal.RequireFromString("0.25")
	definirFator(l, f, decimal.RequireFromString("2.5"), OrigemManual, "")
	nota := &entity.FiscalEntry{Itens: []*entity.FiscalEntryItem{l}}
	if _, err := s.AplicarConversoes(ctx, nota); err != nil {
		t.Fatal(err)
	}
	if !l.FatorConversao.Equal(f) || origem(l) != OrigemManual {
		t.Fatalf("o fator manual foi sobrescrito: %s (%s)", l.FatorConversao, origem(l))
	}
	// Do vínculo: unidade declarada no vínculo tem de casar com a da nota.
	l2 := linhaNota(2, 30, "KG", 10)
	aplicarConversao(l2, decPtr("0.3"), strPtr("PC"))
	if origem(l2) != "" {
		t.Fatalf("vínculo cadastrado para PC não vale para KG: %q", origem(l2))
	}
	aplicarConversao(l2, decPtr("0.3"), strPtr("kg"))
	if origem(l2) != OrigemVinculoFornecedor || l2.QuantidadeEstoque.StringFixed(1) != "3.0" {
		t.Fatalf("vínculo KG deveria valer: %s (%s)", l2.QuantidadeEstoque, origem(l2))
	}
	// Fator 1 do vínculo sem unidade é o padrão da tabela, não decisão.
	aplicarConversao(l2, decPtr("1"), nil)
	if origem(l2) != "" {
		t.Fatalf("fator 1 sem unidade não é decisão: %q", origem(l2))
	}
}

// Fator invertido: custo por metro 16× o da última entrada → atenção.
func TestEntradaCustoForaDoPadraoAvisa(t *testing.T) {
	ctx := context.Background()
	docs := docsUnidade{
		cadastro: map[int64]repository.ItemCandidate{40: {Code: 40, BusinessCode: "TB", UOM: "M"}},
		// Errado: cadastraram 1 KG = 0,25 M quando é 1 KG = 4 M... e o custo denuncia.
		arestas: map[int64][]convservice.Aresta{40: {aresta("KG", "M", "1", "0.25")}},
		custo:   map[int64]decimal.Decimal{40: decimal.RequireFromString("2.50")},
	}
	s := &EntradaServico{Docs: docs}
	l := linhaNota(1, 40, "KG", 100)
	l.TotalPrice = decimal.RequireFromString("1000") // 10/KG; com 4 M/KG seria 2,50/M
	nota := &entity.FiscalEntry{Itens: []*entity.FiscalEntryItem{l}}
	if _, err := s.AplicarConversoes(ctx, nota); err != nil {
		t.Fatal(err)
	}
	pend, err := s.pendenciasDeUnidade(ctx, nota)
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 1 || pend[0].Nivel != entrada.NivelAtencao || !strings.Contains(pend[0].Mensagem, "invertido") {
		t.Fatalf("esperava atenção de custo: %+v", pend)
	}
	// Com o fator certo, sem aviso.
	docs.arestas[40] = []convservice.Aresta{aresta("KG", "M", "1", "4")}
	if _, err := s.AplicarConversoes(ctx, nota); err != nil {
		t.Fatal(err)
	}
	if pend, _ := s.pendenciasDeUnidade(ctx, nota); len(pend) != 0 {
		t.Fatalf("custo coerente não deveria avisar: %+v", pend)
	}
}

func decPtr(s string) *decimal.Decimal {
	d := decimal.RequireFromString(s)
	return &d
}
