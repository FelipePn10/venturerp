package entrada

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// ParametrosConferencia é o que o ERP sabe para conferir a nota do fornecedor
// (como a conferência de impostos do IntegraNF-e do Focco): a tabela de
// alíquotas, o NCM do cadastro e o pedido de compra.
type ParametrosConferencia struct {
	UFEmpresa string
	// Alíquota de IPI por NCM (fração: 0.05 = 5%).
	AliqIPIPorNCM map[string]decimal.Decimal
	// Alíquota interestadual por origem+destino ("SPPR" → 0.12).
	ICMSInterestadual map[string]decimal.Decimal
	// NCM do item do cadastro, pelo código do item.
	NCMDoItem map[int64]string
	// Linhas de pedido de compra ligadas aos itens, pelo código da linha.
	LinhasPedido map[int64]LinhaPedido
}

// LinhaPedido é a linha do pedido de compra já na unidade de estoque.
type LinhaPedido struct {
	Codigo        int64
	PedidoCodigo  int64
	ItemCode      int64
	PrecoUnitario decimal.Decimal // por unidade de estoque
	SaldoAFaturar decimal.Decimal // em unidade de estoque
	TolerancePct  decimal.Decimal
	Cancelada     bool
}

// Divergencia é uma diferença entre a nota e o que o ERP esperava.
type Divergencia struct {
	Nivel     string `json:"nivel"` // IMPEDE ou ATENCAO
	Item      int    `json:"item"`  // 0 = cabeçalho
	Tipo      string `json:"tipo"`
	Mensagem  string `json:"mensagem"`
	Esperado  string `json:"esperado,omitempty"`
	Informado string `json:"informado,omitempty"`
}

var (
	toleranciaCalculo = decimal.NewFromFloat(0.02)
	toleranciaAliq    = decimal.NewFromFloat(0.0001)
	origensImportadas = map[string]bool{"1": true, "2": true, "3": true, "8": true}
	cstsTributados    = map[string]bool{"00": true, "10": true, "20": true, "70": true, "90": true}
)

// Divergencias confere a nota contra a própria aritmética, as tabelas
// fiscais, o cadastro e o pedido de compra. Nenhuma é decidida aqui como
// bloqueio de aprovação, exceto a linha de pedido cancelada: preço e
// quantidade acima da tolerância viram bloqueio na regra de tolerância de
// compras (camada de aplicação), que é configurável por fornecedor.
func Divergencias(e *entity.FiscalEntry, p ParametrosConferencia) []Divergencia {
	var out []Divergencia
	add := func(nivel string, item int, tipo, esperado, informado, msg string, args ...any) {
		out = append(out, Divergencia{Nivel: nivel, Item: item, Tipo: tipo, Mensagem: fmt.Sprintf(msg, args...), Esperado: esperado, Informado: informado})
	}

	somaProdutos, somaICMS, somaIPI := decimal.Zero, decimal.Zero, decimal.Zero
	for _, it := range e.Itens {
		qtd := decimal.NewFromFloat(it.Quantity)
		unit := decimal.NewFromFloat(it.UnitPrice)
		total := decimal.NewFromFloat(it.TotalPrice)
		somaProdutos = somaProdutos.Add(total)
		somaICMS = somaICMS.Add(decimal.NewFromFloat(it.ValorICMS))
		somaIPI = somaIPI.Add(decimal.NewFromFloat(it.ValorIPI))

		// 1. Aritmética da linha (qCom × vUnCom = vProd, com folga de centavo
		// por causa do arredondamento do unitário em até 10 casas).
		if calc := qtd.Mul(unit).Round(2); calc.Sub(total).Abs().GreaterThan(toleranciaCalculo.Add(qtd.Mul(decimal.NewFromFloat(0.00005)))) {
			add(NivelAtencao, it.Sequence, "CALCULO_ITEM", Moeda(calc), Moeda(total),
				"item %d: quantidade × valor unitário dá %s, mas a nota traz %s", it.Sequence, Moeda(calc), Moeda(total))
		}
		// 2. ICMS: base × alíquota.
		if it.ValorICMS > 0 || it.BaseICMS > 0 {
			calc := decimal.NewFromFloat(it.BaseICMS).Mul(decimal.NewFromFloat(it.AliqICMS)).Round(2)
			if calc.Sub(decimal.NewFromFloat(it.ValorICMS)).Abs().GreaterThan(toleranciaCalculo) {
				add(NivelAtencao, it.Sequence, "CALCULO_ICMS", Moeda(calc), Moeda(decimal.NewFromFloat(it.ValorICMS)),
					"item %d: base × alíquota do ICMS dá %s, a nota traz %s", it.Sequence, Moeda(calc), Moeda(decimal.NewFromFloat(it.ValorICMS)))
			}
		}
		// 3. IPI: base × alíquota.
		if it.ValorIPI > 0 || it.BaseIPI > 0 {
			calc := decimal.NewFromFloat(it.BaseIPI).Mul(decimal.NewFromFloat(it.AliqIPI)).Round(2)
			if calc.Sub(decimal.NewFromFloat(it.ValorIPI)).Abs().GreaterThan(toleranciaCalculo) {
				add(NivelAtencao, it.Sequence, "CALCULO_IPI", Moeda(calc), Moeda(decimal.NewFromFloat(it.ValorIPI)),
					"item %d: base × alíquota do IPI dá %s, a nota traz %s", it.Sequence, Moeda(calc), Moeda(decimal.NewFromFloat(it.ValorIPI)))
			}
		}
		ncm := ""
		if it.Ncm != nil {
			ncm = soNumeros(*it.Ncm)
		}
		// 4. Alíquota de IPI da tabela de NCM.
		if aliq, ok := p.AliqIPIPorNCM[ncm]; ok && ncm != "" && it.CstIPI != nil && *it.CstIPI == "50" {
			if aliq.Sub(decimal.NewFromFloat(it.AliqIPI)).Abs().GreaterThan(toleranciaAliq) {
				add(NivelAtencao, it.Sequence, "ALIQUOTA_IPI", pct(aliq), pct(decimal.NewFromFloat(it.AliqIPI)),
					"item %d: o NCM %s tem IPI de %s na tabela, a nota usa %s", it.Sequence, ncm, pct(aliq), pct(decimal.NewFromFloat(it.AliqIPI)))
			}
		}
		// 5. Alíquota interestadual de ICMS (regime normal, operação de outro
		// estado). Mercadoria importada (origem 1, 2, 3 ou 8) é 4% (Res. SF 13/2012).
		if e.UFEmitente != nil && p.UFEmpresa != "" && !strings.EqualFold(*e.UFEmitente, p.UFEmpresa) &&
			it.CstICMS != nil && cstsTributados[*it.CstICMS] && it.AliqICMS > 0 {
			esperado, ok := p.ICMSInterestadual[strings.ToUpper(*e.UFEmitente)+strings.ToUpper(p.UFEmpresa)]
			if it.Origem != nil && origensImportadas[*it.Origem] {
				esperado, ok = decimal.NewFromFloat(0.04), true
			}
			if ok && esperado.Sub(decimal.NewFromFloat(it.AliqICMS)).Abs().GreaterThan(toleranciaAliq) {
				add(NivelAtencao, it.Sequence, "ALIQUOTA_ICMS", pct(esperado), pct(decimal.NewFromFloat(it.AliqICMS)),
					"item %d: a alíquota interestadual %s→%s é %s, a nota usa %s", it.Sequence, *e.UFEmitente, p.UFEmpresa, pct(esperado), pct(decimal.NewFromFloat(it.AliqICMS)))
			}
		}
		// 6. NCM da nota × NCM do item do cadastro.
		if it.ItemCode != nil {
			if cad := soNumeros(p.NCMDoItem[*it.ItemCode]); cad != "" && ncm != "" && cad != ncm {
				add(NivelAtencao, it.Sequence, "NCM", cad, ncm,
					"item %d: o NCM da nota (%s) é diferente do NCM do item %d no cadastro (%s) — confira a conciliação ou a classificação fiscal", it.Sequence, ncm, *it.ItemCode, cad)
			}
		}
		// 7. Pedido de compra: linha, item, preço e saldo.
		if it.PurchaseOrderItemCode != nil {
			l, ok := p.LinhasPedido[*it.PurchaseOrderItemCode]
			switch {
			case !ok:
				add(NivelImpede, it.Sequence, "PEDIDO", "", fmt.Sprint(*it.PurchaseOrderItemCode),
					"item %d: a linha de pedido de compra informada não existe ou não é deste fornecedor", it.Sequence)
			case l.Cancelada:
				add(NivelImpede, it.Sequence, "PEDIDO", "", fmt.Sprint(l.Codigo),
					"item %d: a linha do pedido %d está cancelada", it.Sequence, l.PedidoCodigo)
			default:
				if it.ItemCode != nil && *it.ItemCode != l.ItemCode {
					add(NivelImpede, it.Sequence, "PEDIDO_ITEM", fmt.Sprint(l.ItemCode), fmt.Sprint(*it.ItemCode),
						"item %d: a linha do pedido %d é do item %d, mas a nota foi conciliada com o item %d", it.Sequence, l.PedidoCodigo, l.ItemCode, *it.ItemCode)
				}
				qtdEstoque := qtd
				if it.QuantidadeEstoque != nil {
					qtdEstoque = *it.QuantidadeEstoque
				}
				if qtdEstoque.IsPositive() && l.PrecoUnitario.IsPositive() {
					unitNota := CustoUnitarioBruto(it, qtdEstoque)
					limite := l.PrecoUnitario.Mul(decimal.NewFromInt(1).Add(l.TolerancePct.Div(cem)))
					if unitNota.Sub(limite).GreaterThan(decimal.NewFromFloat(0.0001)) {
						add(NivelAtencao, it.Sequence, "PEDIDO_PRECO", l.PrecoUnitario.StringFixed(4), unitNota.StringFixed(4),
							"item %d: preço unitário %s acima do pedido %d (%s)", it.Sequence, Moeda(unitNota), l.PedidoCodigo, Moeda(l.PrecoUnitario))
					}
				}
				limiteQtd := l.SaldoAFaturar.Mul(decimal.NewFromInt(1).Add(l.TolerancePct.Div(cem)))
				if qtdEstoque.Sub(limiteQtd).GreaterThan(decimal.NewFromFloat(0.0001)) {
					add(NivelAtencao, it.Sequence, "PEDIDO_QUANTIDADE", l.SaldoAFaturar.String(), qtdEstoque.String(),
						"item %d: a nota fatura %s, mas o saldo a faturar do pedido %d é %s", it.Sequence, qtdEstoque.String(), l.PedidoCodigo, l.SaldoAFaturar.String())
				}
			}
		}
	}
	// 8. Totais do cabeçalho × soma dos itens.
	if v := decimal.NewFromFloat(e.ValorProdutos); somaProdutos.Sub(v).Abs().GreaterThan(toleranciaCalculo) {
		add(NivelAtencao, 0, "TOTAL_PRODUTOS", Moeda(somaProdutos), Moeda(v), "a soma dos itens (%s) não fecha com o total de produtos da nota (%s)", Moeda(somaProdutos), Moeda(v))
	}
	if v := decimal.NewFromFloat(e.ValorICMS); somaICMS.Sub(v).Abs().GreaterThan(toleranciaCalculo) {
		add(NivelAtencao, 0, "TOTAL_ICMS", Moeda(somaICMS), Moeda(v), "o ICMS dos itens (%s) não fecha com o ICMS da nota (%s)", Moeda(somaICMS), Moeda(v))
	}
	if v := decimal.NewFromFloat(e.ValorIPI); somaIPI.Sub(v).Abs().GreaterThan(toleranciaCalculo) {
		add(NivelAtencao, 0, "TOTAL_IPI", Moeda(somaIPI), Moeda(v), "o IPI dos itens (%s) não fecha com o IPI da nota (%s)", Moeda(somaIPI), Moeda(v))
	}
	return out
}

// CustoUnitarioBruto é o preço por unidade de estoque como o pedido o
// enxerga: valor do produto menos desconto, sem impostos.
func CustoUnitarioBruto(it *entity.FiscalEntryItem, qtdEstoque decimal.Decimal) decimal.Decimal {
	if !qtdEstoque.IsPositive() {
		return decimal.Zero
	}
	return decimal.NewFromFloat(it.TotalPrice).Sub(it.ValorDesconto).Div(qtdEstoque).Round(6)
}

func pct(v decimal.Decimal) string {
	return strings.Replace(v.Mul(cem).StringFixed(2), ".", ",", 1) + "%"
}
