package entrada

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// CFOPEntrada é o CFOP com que a NOSSA empresa escritura a entrada. O CFOP do
// XML é o do fornecedor (5xxx/6xxx/7xxx, saída dele); o livro de entradas e o
// SPED pedem o de entrada (1xxx/2xxx/3xxx).
//
// Com tipo de operação de entrada (a "natureza"), vale a natureza dele, com o
// primeiro dígito ajustado à localização do fornecedor. Sem ela, o CFOP do
// fornecedor é convertido pela regra usual (5→1, 6→2, 7→3), com as
// equivalências que não são só a troca do dígito (revenda com ST).
func CFOPEntrada(cfopFornecedor, natureza, ufFornecedor, ufEmpresa string) string {
	digitoLocal := func(padrao byte) byte {
		uf := strings.ToUpper(strings.TrimSpace(ufFornecedor))
		switch {
		case uf == "EX":
			return '3'
		case uf != "" && strings.EqualFold(uf, strings.TrimSpace(ufEmpresa)):
			return '1'
		case uf != "" && strings.TrimSpace(ufEmpresa) != "":
			return '2'
		}
		return padrao
	}
	if n := soNumeros(natureza); len(n) == 4 {
		b := []byte(n)
		b[0] = digitoLocal(b[0])
		return string(b)
	}
	c := soNumeros(cfopFornecedor)
	if len(c) != 4 {
		return ""
	}
	var d byte
	switch c[0] {
	case '5':
		d = '1'
	case '6':
		d = '2'
	case '7':
		d = '3'
	case '1', '2', '3':
		return c // já é de entrada (nota de entrada própria/devolução)
	default:
		return ""
	}
	// A UF de quem emite e de quem recebe decide dentro/fora do estado; o
	// primeiro dígito do CFOP do fornecedor só vale quando falta uma das UFs
	// (nota com 6xxx entre empresas do mesmo estado é erro do emitente).
	d = digitoLocal(d)
	resto := c[1:]
	switch resto {
	case "401", "403", "405": // venda com ST → compra com ST
		resto = "403"
	case "656", "655": // venda de combustível → compra
		resto = "652"
	}
	return string(d) + resto
}

// CustoAquisicao é o valor que entra no estoque/custo: o valor contábil menos
// os tributos que a empresa recupera como crédito. ICMS, PIS e COFINS estão
// dentro do preço; IPI e ICMS-ST foram somados ao valor contábil.
func CustoAquisicao(it *entity.FiscalEntryItem) decimal.Decimal {
	c := it.ValorContabil
	if it.GeraCreditoICMS {
		c = c.Sub(decimal.NewFromFloat(it.ValorICMS))
	}
	if it.GeraCreditoIPI {
		c = c.Sub(decimal.NewFromFloat(it.ValorIPI))
	}
	if it.GeraCreditoPIS {
		c = c.Sub(decimal.NewFromFloat(it.ValorPIS))
	}
	if it.GeraCreditoCOFINS {
		c = c.Sub(decimal.NewFromFloat(it.ValorCOFINS))
	}
	if it.GeraCreditoIBSCBS {
		c = c.Sub(it.ValorIBS).Sub(it.ValorCBS)
	}
	if c.IsNegative() {
		return decimal.Zero
	}
	return c.Round(2)
}

// PlanoFinanceiro diz quanto da nota vai ao contas a pagar do fornecedor e
// quanto cada plano de contas deve receber nas parcelas.
//
//   - base: valor contábil dos itens que geram financeiro (bonificação,
//     comodato e remessa não geram);
//   - a pagar: a base menos as retenções (o fornecedor recebe o líquido; a
//     retenção vira título do imposto a recolher);
//   - alvos: o total de cada plano escalado para fechar exatamente no valor a
//     pagar (a diferença de cada plano vai para os títulos de retenção).
func PlanoFinanceiro(e *entity.FiscalEntry) (base, aPagar decimal.Decimal, alvos, residuo map[entity.ChaveConta]decimal.Decimal) {
	financeiros := make([]*entity.FiscalEntryItem, 0, len(e.Itens))
	for _, it := range e.Itens {
		if it.GeraFinanceiro {
			financeiros = append(financeiros, it)
			base = base.Add(it.ValorContabil)
		}
	}
	base = base.Round(2)
	aPagar = base.Sub(e.TotalRetencoes())
	if aPagar.IsNegative() {
		aPagar = decimal.Zero
	}
	totais := TotalPorConta(financeiros)
	alvos = Escalar(totais, aPagar)
	residuo = map[entity.ChaveConta]decimal.Decimal{}
	for k, v := range totais {
		if r := v.Sub(alvos[k]); !r.IsZero() {
			residuo[k] = r
		}
	}
	return base, aPagar, alvos, residuo
}

// Escalar redistribui os totais para somarem exatamente alvo, mantendo a
// proporção (a última conta absorve os centavos). Sem diferença, devolve os
// próprios totais.
func Escalar(totais map[entity.ChaveConta]decimal.Decimal, alvo decimal.Decimal) map[entity.ChaveConta]decimal.Decimal {
	soma := decimal.Zero
	for _, v := range totais {
		soma = soma.Add(v)
	}
	out := map[entity.ChaveConta]decimal.Decimal{}
	if soma.Equal(alvo) || !soma.IsPositive() {
		for k, v := range totais {
			out[k] = v
		}
		return out
	}
	contas := ChavesOrdenadas(totais)
	acumulado := decimal.Zero
	for i, k := range contas {
		if i == len(contas)-1 {
			out[k] = alvo.Sub(acumulado)
			continue
		}
		v := totais[k].Mul(alvo).Div(soma).Round(2)
		out[k] = v
		acumulado = acumulado.Add(v)
	}
	return out
}

// Retencao é um imposto retido do fornecedor que a empresa recolhe.
type Retencao struct {
	Tipo       string // PIS, COFINS, CSLL, IRRF, INSS, ISS
	Descricao  string
	Valor      decimal.Decimal
	Vencimento time.Time
}

// Retencoes lista os títulos de imposto a recolher que a nota gera, com o
// vencimento legal: PIS/COFINS/CSLL, IRRF e INSS até o dia 20 do mês seguinte
// ao fato gerador (antecipado se não for dia útil); ISS no dia 10 do mês
// seguinte (regra mais comum dos municípios — confira a do seu).
func Retencoes(e *entity.FiscalEntry) []Retencao {
	ref := e.DataEmissao
	dia20 := antecipaFimDeSemana(diaDoMesSeguinte(ref, 20))
	dia10 := antecipaFimDeSemana(diaDoMesSeguinte(ref, 10))
	cand := []Retencao{
		{"PIS", "PIS retido na fonte", e.ValorRetPIS, dia20},
		{"COFINS", "COFINS retida na fonte", e.ValorRetCOFINS, dia20},
		{"CSLL", "CSLL retida na fonte", e.ValorRetCSLL, dia20},
		{"IRRF", "IRRF retido na fonte", e.ValorIRRF, dia20},
		{"INSS", "INSS retido (previdência)", e.ValorRetPrev, dia20},
		{"ISS", "ISS retido", e.ValorISSRet, dia10},
	}
	out := make([]Retencao, 0, len(cand))
	for _, r := range cand {
		if r.Valor.IsPositive() {
			r.Valor = r.Valor.Round(2)
			out = append(out, r)
		}
	}
	return out
}

func diaDoMesSeguinte(t time.Time, dia int) time.Time {
	primeiro := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	return primeiro.AddDate(0, 0, dia-1)
}

func antecipaFimDeSemana(t time.Time) time.Time {
	switch t.Weekday() {
	case time.Saturday:
		return t.AddDate(0, 0, -1)
	case time.Sunday:
		return t.AddDate(0, 0, -2)
	}
	return t
}

func soNumeros(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
