// Package entrada reúne as regras da nota fiscal de entrada que não dependem
// de banco: totais por plano de contas, distribuição das parcelas e a
// conferência que libera a aprovação.
package entrada

import (
	"fmt"
	"sort"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// Tolerancia é a diferença aceita entre somas que deveriam fechar (meio
// centavo de arredondamento por linha não é divergência).
var Tolerancia = decimal.NewFromFloat(0.009)

var cem = decimal.NewFromInt(100)

// TotalPorConta soma o valor contábil dos itens por plano de contas/centro de
// custo. Itens sem plano de contas ficam de fora (e são acusados como
// pendência pela conferência).
func TotalPorConta(itens []*entity.FiscalEntryItem) map[entity.ChaveConta]decimal.Decimal {
	out := map[entity.ChaveConta]decimal.Decimal{}
	for _, it := range itens {
		k := it.ChaveConta()
		if k == nil {
			continue
		}
		out[*k] = out[*k].Add(it.ValorContabil)
	}
	return out
}

// ChavesOrdenadas devolve as contas em ordem estável (por plano e centro de
// custo), para o rateio ser sempre o mesmo para a mesma nota.
func ChavesOrdenadas(m map[entity.ChaveConta]decimal.Decimal) []entity.ChaveConta {
	ks := make([]entity.ChaveConta, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool {
		if ks[i].PlanoContasID != ks[j].PlanoContasID {
			return ks[i].PlanoContasID < ks[j].PlanoContasID
		}
		return ks[i].CentroCustoID < ks[j].CentroCustoID
	})
	return ks
}

// AjustarValorContabil faz a soma dos itens fechar exatamente com o total da
// nota. A nota pode trazer componentes que não estão por item (II, serviços,
// arredondamentos do emissor); a diferença vai para o item de maior valor,
// para não sumir dinheiro entre a nota e o contas a pagar.
func AjustarValorContabil(itens []*entity.FiscalEntryItem, totalNota decimal.Decimal) {
	if len(itens) == 0 || !totalNota.IsPositive() {
		return
	}
	soma := decimal.Zero
	maior := 0
	for i, it := range itens {
		it.ValorContabil = it.ValorContabil.Round(2)
		soma = soma.Add(it.ValorContabil)
		if it.ValorContabil.GreaterThan(itens[maior].ValorContabil) {
			maior = i
		}
	}
	dif := totalNota.Round(2).Sub(soma)
	if dif.IsZero() {
		return
	}
	novo := itens[maior].ValorContabil.Add(dif)
	if novo.IsNegative() {
		return
	}
	itens[maior].ValorContabil = novo
}

// DistribuirProporcional preenche a distribuição de cada parcela na
// proporção do total de cada conta sobre o total das parcelas. É o padrão: uma
// nota de 100 mil com 50 mil de matéria-prima e 50 mil de EPI em 3 parcelas
// fica com cada parcela meio a meio. O usuário pode mudar depois (ex.:
// primeira parcela toda em EPI).
//
// O arredondamento é fechado em duas direções: cada conta soma o seu total (a
// última parcela absorve os centavos) e, quando todos os itens já estão
// classificados, cada parcela soma o seu valor (a última conta absorve os
// centavos). Com parte dos itens ainda sem plano, a parcela fica distribuída
// só na parte já classificada — o resto aparece como pendência, em vez de ser
// empurrado para os planos que já existem.
func DistribuirProporcional(parcelas []*entity.FiscalEntryInstallment, totais map[entity.ChaveConta]decimal.Decimal) error {
	if len(parcelas) == 0 {
		return nil
	}
	contas := ChavesOrdenadas(totais)
	if len(contas) == 0 {
		for _, p := range parcelas {
			p.Distribuicao = nil
		}
		return nil
	}
	totalClassificado := decimal.Zero
	for _, k := range contas {
		totalClassificado = totalClassificado.Add(totais[k])
	}
	totalParcelas := decimal.Zero
	for _, p := range parcelas {
		totalParcelas = totalParcelas.Add(p.Valor)
	}
	if !totalClassificado.IsPositive() {
		return fmt.Errorf("os itens classificados não têm valor para distribuir nas parcelas")
	}
	base := decimal.Max(totalParcelas, totalClassificado)
	fechaLinha := totalClassificado.Sub(totalParcelas).Abs().LessThanOrEqual(Tolerancia)

	acumulado := map[entity.ChaveConta]decimal.Decimal{}
	for pi, p := range parcelas {
		ultimaParcela := pi == len(parcelas)-1
		dist := make([]entity.InstallmentAllocation, 0, len(contas))
		somaLinha := decimal.Zero
		for ci, k := range contas {
			var v decimal.Decimal
			switch {
			case ultimaParcela:
				v = totais[k].Sub(acumulado[k])
			case fechaLinha && ci == len(contas)-1:
				v = p.Valor.Sub(somaLinha)
			default:
				v = p.Valor.Mul(totais[k]).Div(base).Round(2)
			}
			if v.IsNegative() {
				v = decimal.Zero
			}
			acumulado[k] = acumulado[k].Add(v)
			somaLinha = somaLinha.Add(v)
			dist = append(dist, entity.InstallmentAllocation{
				PlanoContasID: k.PlanoContasID,
				CentroCustoID: k.CentroCusto(),
				Valor:         v,
			})
		}
		p.Distribuicao = dist
	}
	return nil
}

// Pendencia é o que impede (ou só alerta sobre) a aprovação da nota.
type Pendencia struct {
	Nivel    string `json:"nivel"` // IMPEDE ou ATENCAO
	Campo    string `json:"campo"`
	Mensagem string `json:"mensagem"`
}

const (
	NivelImpede  = "IMPEDE"
	NivelAtencao = "ATENCAO"
)

// Conferir é a conferência de uma nota sem retenções e com todos os itens
// gerando financeiro: as parcelas fecham com o total e cada plano com os seus
// itens. A nota completa usa ConferirEntrada.
func Conferir(totalNota decimal.Decimal, itens []*entity.FiscalEntryItem, parcelas []*entity.FiscalEntryInstallment, semPagamento bool) []Pendencia {
	return conferir(totalNota, TotalPorConta(itens), itens, parcelas, semPagamento)
}

// ConferirEntrada devolve as pendências da nota de entrada. A aprovação só
// passa sem nenhuma de nível IMPEDE:
//
//   - o emitente cadastrado como fornecedor;
//   - todo item conciliado com um item do cadastro e com plano de contas;
//   - todo item que movimenta estoque com almoxarifado e CFOP de entrada;
//   - parcelas somando o valor a pagar (itens que geram financeiro menos as
//     retenções);
//   - cada parcela distribuída por inteiro entre os planos de contas;
//   - cada plano de contas recebendo, somadas as parcelas, a sua parte do
//     valor a pagar.
func ConferirEntrada(e *entity.FiscalEntry) []Pendencia {
	_, aPagar, alvos, _ := PlanoFinanceiro(e)
	out := conferir(aPagar, alvos, e.Itens, e.Parcelas, e.SemPagamento || !aPagar.IsPositive())
	if e.SupplierCode == nil {
		out = append([]Pendencia{{Nivel: NivelImpede, Campo: "supplier_code",
			Mensagem: fmt.Sprintf("o emitente %s (%s) não está cadastrado como fornecedor: cadastre-o a partir da nota ou em VSUP0500 — sem ele o contas a pagar, o pedido de compra e a devolução não têm a quem se ligar",
				e.CnpjEmitente, e.RazaoSocialEmitente)}}, out...)
	}
	for _, it := range e.Itens {
		if it.ItemCode == nil || !it.MovimentaEstoque {
			continue
		}
		desc := ""
		if it.Description != nil {
			desc = " (" + *it.Description + ")"
		}
		if it.WarehouseID == nil {
			out = append(out, Pendencia{Nivel: NivelImpede, Campo: fmt.Sprintf("itens[%d].warehouse_id", it.Sequence),
				Mensagem: fmt.Sprintf("item %d%s movimenta estoque e está sem almoxarifado", it.Sequence, desc)})
		}
	}
	for _, it := range e.Itens {
		if it.CfopEntrada == nil || len(*it.CfopEntrada) != 4 {
			out = append(out, Pendencia{Nivel: NivelAtencao, Campo: fmt.Sprintf("itens[%d].cfop_entrada", it.Sequence),
				Mensagem: fmt.Sprintf("item %d sem CFOP de entrada: o livro de entradas sai com o CFOP do fornecedor", it.Sequence)})
		}
	}
	if r := e.TotalRetencoes(); r.IsPositive() {
		out = append(out, Pendencia{Nivel: NivelAtencao, Campo: "retencoes",
			Mensagem: fmt.Sprintf("a nota tem %s de retenções: o fornecedor recebe %s e a empresa recolhe a diferença (títulos de imposto gerados na aprovação)", Moeda(r), Moeda(aPagar))})
	}
	return out
}

func conferir(totalAPagar decimal.Decimal, totais map[entity.ChaveConta]decimal.Decimal, itens []*entity.FiscalEntryItem, parcelas []*entity.FiscalEntryInstallment, semPagamento bool) []Pendencia {
	var out []Pendencia
	impede := func(campo, msg string, args ...any) {
		out = append(out, Pendencia{Nivel: NivelImpede, Campo: campo, Mensagem: fmt.Sprintf(msg, args...)})
	}
	atencao := func(campo, msg string, args ...any) {
		out = append(out, Pendencia{Nivel: NivelAtencao, Campo: campo, Mensagem: fmt.Sprintf(msg, args...)})
	}

	if len(itens) == 0 {
		impede("itens", "a nota não tem itens")
	}
	for _, it := range itens {
		desc := ""
		if it.Description != nil {
			desc = " (" + *it.Description + ")"
		}
		if it.ItemCode == nil {
			impede(fmt.Sprintf("itens[%d].item_code", it.Sequence),
				"item %d%s não está conciliado com um item do cadastro", it.Sequence, desc)
		}
		if it.PlanoContasID == nil {
			impede(fmt.Sprintf("itens[%d].plano_contas_id", it.Sequence),
				"item %d%s está sem plano de contas", it.Sequence, desc)
		}
	}

	if semPagamento {
		if len(parcelas) > 0 {
			atencao("parcelas", "a nota declara que não há pagamento (bonificação/remessa), mas tem parcelas: elas vão gerar contas a pagar")
		}
	} else if len(parcelas) == 0 && totalAPagar.IsPositive() {
		impede("parcelas", "informe ao menos uma parcela: sem ela a nota não gera contas a pagar")
	}
	if len(parcelas) == 0 {
		return out
	}

	somaParcelas := decimal.Zero
	for _, p := range parcelas {
		somaParcelas = somaParcelas.Add(p.Valor)
	}
	if somaParcelas.Sub(totalAPagar).Abs().GreaterThan(Tolerancia) {
		impede("parcelas", "as parcelas somam %s e o valor a pagar ao fornecedor é %s", Moeda(somaParcelas), Moeda(totalAPagar))
	}

	nome := nomesDasContas(itens)
	semPlano := 0
	for _, it := range itens {
		if it.PlanoContasID == nil {
			semPlano++
		}
	}
	if semPlano > 0 && len(totais) > 0 {
		// Item sem plano já é pendência; acusar cada parcela "incompleta" por
		// causa dele só repetiria o mesmo problema.
		impede("parcelas.distribuicao",
			"a distribuição das parcelas só fecha depois que todos os itens tiverem plano de contas (%d sem plano)", semPlano)
	}
	distribuido := map[entity.ChaveConta]decimal.Decimal{}
	for _, p := range parcelas {
		if len(p.Distribuicao) == 0 {
			if len(totais) > 0 {
				impede(fmt.Sprintf("parcelas[%d].distribuicao", p.Numero),
					"a parcela %d não tem distribuição por plano de contas", p.Numero)
			}
			continue
		}
		somaDist := decimal.Zero
		for _, a := range p.Distribuicao {
			if a.Valor.IsNegative() {
				impede(fmt.Sprintf("parcelas[%d].distribuicao", p.Numero), "a parcela %d tem valor negativo na distribuição", p.Numero)
			}
			somaDist = somaDist.Add(a.Valor)
			distribuido[a.Chave()] = distribuido[a.Chave()].Add(a.Valor)
		}
		if semPlano == 0 && somaDist.Sub(p.Valor).Abs().GreaterThan(Tolerancia) {
			impede(fmt.Sprintf("parcelas[%d].distribuicao", p.Numero),
				"a distribuição da parcela %d soma %s, mas a parcela vale %s", p.Numero, Moeda(somaDist), Moeda(p.Valor))
		}
	}
	for _, k := range ChavesOrdenadas(totais) {
		if distribuido[k].Sub(totais[k]).Abs().GreaterThan(Tolerancia) {
			impede("parcelas.distribuicao",
				"o plano de contas %s recebe %s nas parcelas, mas a parte dele no valor a pagar é %s",
				nome(k), Moeda(distribuido[k]), Moeda(totais[k]))
		}
	}
	for k, v := range distribuido {
		if _, ok := totais[k]; !ok && v.IsPositive() {
			impede("parcelas.distribuicao",
				"as parcelas distribuem %s para o plano de contas %s, que não tem nenhum item da nota", Moeda(v), nome(k))
		}
	}
	return out
}

// TemImpedimento diz se alguma pendência bloqueia a aprovação.
func TemImpedimento(ps []Pendencia) bool {
	for _, p := range ps {
		if p.Nivel == NivelImpede {
			return true
		}
	}
	return false
}

// ProporcaoPaga devolve quanto de cada rateio já foi pago, na proporção do
// pagamento do título (usado no realizado por plano de contas).
func ProporcaoPaga(valorRateio, valorPago, valorTitulo decimal.Decimal) decimal.Decimal {
	if !valorTitulo.IsPositive() || !valorPago.IsPositive() {
		return decimal.Zero
	}
	if valorPago.GreaterThanOrEqual(valorTitulo) {
		return valorRateio
	}
	return valorRateio.Mul(valorPago).Div(valorTitulo).Round(2)
}

// Percentual é a participação de parte no todo, em %.
func Percentual(parte, todo decimal.Decimal) decimal.Decimal {
	if !todo.IsPositive() {
		return decimal.Zero
	}
	return parte.Mul(cem).Div(todo).Round(4)
}

// nomesDasContas devolve como cada conta aparece nas mensagens: código e nome
// do plano quando os itens os trazem, o id quando não.
func nomesDasContas(itens []*entity.FiscalEntryItem) func(entity.ChaveConta) string {
	nomes := map[entity.ChaveConta]string{}
	for _, it := range itens {
		k := it.ChaveConta()
		if k == nil || it.PlanoContasCodigo == nil {
			continue
		}
		n := *it.PlanoContasCodigo
		if it.PlanoContasNome != nil && *it.PlanoContasNome != "" {
			n += " " + *it.PlanoContasNome
		}
		if it.CentroCustoNome != nil && *it.CentroCustoNome != "" {
			n += " / " + *it.CentroCustoNome
		}
		nomes[*k] = n
	}
	return func(k entity.ChaveConta) string {
		if n, ok := nomes[k]; ok {
			return n
		}
		return fmt.Sprintf("%d", k.PlanoContasID)
	}
}

// Moeda formata o valor como o usuário lê: R$ 66.666,66.
func Moeda(v decimal.Decimal) string {
	txt := v.Abs().StringFixed(2)
	inteiro, centavos := txt[:len(txt)-3], txt[len(txt)-2:]
	var b []byte
	for i, c := range []byte(inteiro) {
		if i > 0 && (len(inteiro)-i)%3 == 0 {
			b = append(b, '.')
		}
		b = append(b, c)
	}
	sinal := ""
	if v.IsNegative() {
		sinal = "-"
	}
	return "R$ " + sinal + string(b) + "," + centavos
}
