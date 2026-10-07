// Package frete rateia o frete de compra (CT-e da transportadora) entre os
// itens das notas de entrada que ele transportou. É o "conhecimento de frete"
// dos ERPs de mercado: o frete pago à transportadora é custo de aquisição da
// mercadoria — entra no custo médio do que ainda está no estoque e vai para
// despesa a parte do que já foi consumido.
package frete

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

const (
	RateioValor      = "VALOR"
	RateioQuantidade = "QUANTIDADE"
	RateioPeso       = "PESO"
)

// Item é a linha de nota de entrada que recebe parte do frete.
type Item struct {
	ItemID, EntryID  int64
	ItemCode         *int64
	WarehouseID      *int64
	PlanoContasID    *int64
	CentroCustoID    *int64
	Descricao        string
	ValorContabil    decimal.Decimal
	Quantidade       decimal.Decimal // na unidade de estoque
	PesoUnitario     *decimal.Decimal
	MovimentaEstoque bool
	// EmEstoque: quanto do item ainda está no almoxarifado (saldo atual,
	// limitado à quantidade da nota) — define a parte que vira custo.
	EmEstoque decimal.Decimal
}

// Parte é o frete atribuído a um item.
type Parte struct {
	Item         Item
	Valor        decimal.Decimal // custo do frete do item
	ValorEstoque decimal.Decimal // complemento do custo médio
	ValorDespesa decimal.Decimal // parte já consumida
}

// CustoDoFrete: o ICMS do CT-e creditado não é custo.
func CustoDoFrete(valor, icms decimal.Decimal, creditaICMS bool) decimal.Decimal {
	if creditaICMS {
		return valor.Sub(icms)
	}
	return valor
}

func base(tipo string, it Item) (decimal.Decimal, error) {
	switch tipo {
	case RateioValor:
		return it.ValorContabil, nil
	case RateioQuantidade:
		return it.Quantidade, nil
	case RateioPeso:
		if it.PesoUnitario == nil || !it.PesoUnitario.IsPositive() {
			return decimal.Zero, fmt.Errorf("item %q sem peso no cadastro", it.Descricao)
		}
		return it.Quantidade.Mul(*it.PesoUnitario), nil
	}
	return decimal.Zero, fmt.Errorf("tipo de rateio inválido: %s (VALOR, QUANTIDADE ou PESO)", tipo)
}

// Ratear distribui o custo pelos itens na proporção da base escolhida (a
// última parte leva os centavos) e separa, em cada item, o que vai ao estoque
// (proporção ainda no almoxarifado) e o que vai à despesa.
func Ratear(custo decimal.Decimal, tipo string, itens []Item) ([]Parte, error) {
	tipo = strings.ToUpper(strings.TrimSpace(tipo))
	if len(itens) == 0 {
		return nil, fmt.Errorf("o frete não tem itens de nota para ratear: vincule as notas de entrada que ele transportou")
	}
	if !custo.IsPositive() {
		return nil, fmt.Errorf("o custo do frete precisa ser positivo")
	}
	bases := make([]decimal.Decimal, len(itens))
	total := decimal.Zero
	var semPeso []string
	for i, it := range itens {
		b, err := base(tipo, it)
		if err != nil {
			if tipo == RateioPeso {
				semPeso = append(semPeso, it.Descricao)
				continue
			}
			return nil, err
		}
		if b.IsNegative() {
			b = decimal.Zero
		}
		bases[i] = b
		total = total.Add(b)
	}
	if len(semPeso) > 0 {
		return nil, fmt.Errorf("rateio por peso: itens sem peso no cadastro — %s", strings.Join(semPeso, "; "))
	}
	if !total.IsPositive() {
		return nil, fmt.Errorf("a base do rateio por %s é zero", strings.ToLower(tipo))
	}
	out := make([]Parte, len(itens))
	acumulado := decimal.Zero
	for i, it := range itens {
		v := custo.Mul(bases[i]).Div(total).Round(2)
		if i == len(itens)-1 {
			v = custo.Sub(acumulado)
		}
		acumulado = acumulado.Add(v)
		p := Parte{Item: it, Valor: v}
		if it.MovimentaEstoque && it.ItemCode != nil && it.WarehouseID != nil && it.Quantidade.IsPositive() {
			fr := decimal.Min(decimal.Max(it.EmEstoque, decimal.Zero), it.Quantidade).Div(it.Quantidade)
			p.ValorEstoque = v.Mul(fr).Round(2)
		}
		p.ValorDespesa = v.Sub(p.ValorEstoque)
		out[i] = p
	}
	return out, nil
}

// PesoEmKg converte o peso do cadastro (unidade KG, G ou T) para quilos.
func PesoEmKg(peso decimal.Decimal, unidade string) decimal.Decimal {
	switch strings.ToUpper(strings.TrimSpace(unidade)) {
	case "G":
		return peso.Div(decimal.NewFromInt(1000))
	case "T", "TON":
		return peso.Mul(decimal.NewFromInt(1000))
	}
	return peso
}
