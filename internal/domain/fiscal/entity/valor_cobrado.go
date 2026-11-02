package entity

import (
	"strings"

	"github.com/shopspring/decimal"
)

// RetornoDeIndustrializacao diz se o CFOP devolve mercadoria recebida para
// industrialização por encomenda: x.902 (material aplicado) e x.903 (sobra,
// perda e sucata não aplicadas). O material é do CLIENTE — volta na mesma nota
// em que a empresa cobra o serviço, mas não é venda: não vira duplicata, título
// nem receita.
func RetornoDeIndustrializacao(cfop string) bool {
	c := strings.ReplaceAll(strings.TrimSpace(cfop), ".", "")
	if len(c) != 4 || (c[0] != '5' && c[0] != '6') {
		return false
	}
	return c[1:] == "902" || c[1:] == "903"
}

// ValorCobrado é o que o cliente PAGA pela nota: o total do documento menos o
// material dele que está voltando.
//
// A NF-e 5.911 da Usimac é o caso: serviço R$ 230,00 (5124) + material
// retornando R$ 182,60 (5902) = R$ 412,60 de total da nota, e a fatura é de
// R$ 230,00. Parcelar sobre o total cobraria do cliente o próprio material.
//
// Sem linha de retorno, é o total da nota — nada muda para as demais notas.
func ValorCobrado(exit *FiscalExit, items []*FiscalExitItem) decimal.Decimal {
	if exit == nil {
		return decimal.Zero
	}
	total := decimal.NewFromFloat(exit.ValorTotal)
	for _, it := range items {
		if it != nil && RetornoDeIndustrializacao(it.Cfop) {
			total = total.Sub(decimal.NewFromFloat(it.TotalPrice))
		}
	}
	if total.IsNegative() {
		return decimal.Zero
	}
	return total.Round(2)
}
