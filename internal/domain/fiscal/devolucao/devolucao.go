// Package devolucao monta a NF-e de devolução de compra: os itens devolvidos
// ao fornecedor com os mesmos valores e impostos da nota de entrada, na
// proporção da quantidade devolvida, e o CFOP de devolução correspondente ao
// CFOP de entrada.
package devolucao

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// sufixos: CFOP de entrada (3 últimos dígitos) → CFOP de devolução.
var sufixos = map[string]string{
	"101": "201", // compra para industrialização
	"102": "202", // compra para comercialização
	"111": "201", "113": "202", "116": "201", "117": "202", "118": "202", "120": "201", "121": "202", "122": "201",
	"401": "410",                             // compra para industrialização com ST
	"403": "411",                             // compra para comercialização com ST
	"406": "412",                             // ativo imobilizado com ST
	"407": "413",                             // uso e consumo com ST
	"551": "553",                             // ativo imobilizado
	"556": "556",                             // uso e consumo
	"651": "660", "652": "661", "653": "662", // combustíveis
}

// CFOP devolve o CFOP de devolução para o CFOP de entrada do item. O primeiro
// dígito vem da operação (5 dentro do estado, 6 fora, 7 exterior); CFOP de
// entrada sem correspondência cai em x949 (outra saída não especificada).
func CFOP(cfopEntrada string, mesmaUF, exterior bool) string {
	d := "6"
	if mesmaUF {
		d = "5"
	}
	if exterior {
		d = "7"
	}
	c := strings.TrimSpace(cfopEntrada)
	if len(c) == 4 {
		if s, ok := sufixos[c[1:]]; ok {
			return d + s
		}
	}
	return d + "949"
}

// Origem é a linha da nota de entrada.
type Origem struct {
	ItemID                                           int64
	Descricao                                        string
	Quantidade                                       decimal.Decimal // da nota (unidade comercial)
	Devolvida                                        decimal.Decimal // já em notas de devolução não canceladas
	ValorUnitario                                    decimal.Decimal
	Total, Frete, Seguro, Desconto, Outras           decimal.Decimal
	BaseICMS, ICMS, BaseIPI, IPI, PIS, COFINS        decimal.Decimal
	BaseST, ST                                       decimal.Decimal
	CustoAquisicao                                   decimal.Decimal
	CreditoICMS, CreditoIPI, CreditoPIS, CreditoCOFI decimal.Decimal // créditos que a entrada aproveitou
}

// Linha é a linha da devolução, proporcional à quantidade devolvida.
type Linha struct {
	Origem                                    Origem
	Quantidade                                decimal.Decimal
	Total, Frete, Seguro, Desconto, Outras    decimal.Decimal
	BaseICMS, ICMS, BaseIPI, IPI, PIS, COFINS decimal.Decimal
	BaseST, ST                                decimal.Decimal
	// CustoAquisicao e os créditos estornados da parte devolvida.
	CustoAquisicao                                   decimal.Decimal
	CreditoICMS, CreditoIPI, CreditoPIS, CreditoCOFI decimal.Decimal
}

// ValorContabil da linha: o que o fornecedor deve de volta por ela.
func (l Linha) ValorContabil() decimal.Decimal {
	return l.Total.Add(l.Frete).Add(l.Seguro).Add(l.Outras).Sub(l.Desconto).Add(l.IPI).Add(l.ST)
}

// Disponivel: o que ainda pode ser devolvido do item.
func (o Origem) Disponivel() decimal.Decimal {
	d := o.Quantidade.Sub(o.Devolvida)
	if d.IsNegative() {
		return decimal.Zero
	}
	return d
}

// Montar calcula a linha na proporção da quantidade (valores arredondados a
// centavos); devolver a quantidade inteira da nota leva o total exato.
func Montar(o Origem, qtd decimal.Decimal) (Linha, error) {
	if !qtd.IsPositive() {
		return Linha{}, fmt.Errorf("item %q: informe a quantidade devolvida", o.Descricao)
	}
	if qtd.GreaterThan(o.Disponivel()) {
		return Linha{}, fmt.Errorf("item %q: quer devolver %s, mas só restam %s (da nota: %s, já devolvido: %s)",
			o.Descricao, qtd.String(), o.Disponivel().String(), o.Quantidade.String(), o.Devolvida.String())
	}
	if !o.Quantidade.IsPositive() {
		return Linha{}, fmt.Errorf("item %q sem quantidade na nota", o.Descricao)
	}
	f := qtd.Div(o.Quantidade)
	p := func(v decimal.Decimal) decimal.Decimal { return v.Mul(f).Round(2) }
	l := Linha{
		Origem: o, Quantidade: qtd,
		Total: p(o.Total), Frete: p(o.Frete), Seguro: p(o.Seguro), Desconto: p(o.Desconto), Outras: p(o.Outras),
		BaseICMS: p(o.BaseICMS), ICMS: p(o.ICMS), BaseIPI: p(o.BaseIPI), IPI: p(o.IPI), PIS: p(o.PIS), COFINS: p(o.COFINS),
		BaseST: p(o.BaseST), ST: p(o.ST), CustoAquisicao: p(o.CustoAquisicao),
		CreditoICMS: p(o.CreditoICMS), CreditoIPI: p(o.CreditoIPI), CreditoPIS: p(o.CreditoPIS), CreditoCOFI: p(o.CreditoCOFI),
	}
	if qtd.Equal(o.Quantidade) {
		l.Total = o.Total
	}
	return l, nil
}

// cstTabelaB são as tributações do ICMS do regime normal (tabela B do CST).
var cstTabelaB = map[string]bool{"00": true, "02": true, "10": true, "15": true, "20": true, "30": true, "40": true, "41": true,
	"50": true, "51": true, "53": true, "60": true, "61": true, "70": true, "90": true}

// CSTICMS é o CST do ICMS da NF-e de devolução, emitida pela EMPRESA (não pelo
// fornecedor): a nota de compra traz o CST de quem vendeu — CSOSN quando o
// fornecedor é do Simples —, e a SEFAZ recusa CSOSN de emitente do regime
// normal (591) e CST de emitente do Simples. Regime normal mantém a tributação
// da compra quando é da tabela B e usa 90 (outras) quando ela veio em CSOSN;
// Simples Nacional usa o CSOSN 900.
func CSTICMS(cstCompra string, emitenteSimples bool) string {
	if emitenteSimples {
		return "900"
	}
	c := strings.TrimSpace(cstCompra)
	if len(c) == 3 && cstTabelaB[c[1:]] && c[0] >= '0' && c[0] <= '8' && !csosn[c] {
		c = c[1:]
	}
	if cstTabelaB[c] {
		return c
	}
	return "90"
}

// csosn: códigos do Simples Nacional (tabela de CSOSN).
var csosn = map[string]bool{"101": true, "102": true, "103": true, "201": true, "202": true, "203": true, "300": true, "400": true, "500": true, "900": true}
