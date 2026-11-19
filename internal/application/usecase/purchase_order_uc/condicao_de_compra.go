package purchase_order_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
)

// CondicaoDeCompra é como o item é comprado daquele fornecedor: a unidade de
// compra (fonte de suprimento do fornecedor → cadastro do item → estoque), as
// casas decimais dela no catálogo e o preço vigente da tabela de compra.
type CondicaoDeCompra struct {
	// Rotulo é o código de negócio do item, para as mensagens.
	Rotulo  string
	Unidade string
	// Casas da unidade de compra no catálogo; -1 quando não se sabe.
	Casas int32
	// Precos: as linhas vigentes da tabela de compra para o item, na ordem de
	// preferência (do fornecedor antes da geral, faixa maior antes da menor).
	// A faixa (MinQtd) está na unidade da própria linha da tabela.
	Precos []PrecoDeTabela
}

// PrecoDeTabela é uma linha vigente da tabela de preço de compra.
type PrecoDeTabela struct {
	Preco   decimal.Decimal
	Unidade string // vazia = unidade de estoque
	MinQtd  decimal.Decimal
	Tabela  int64
}

// LeitorDeCondicaoDeCompra lê a condição de compra do item no fornecedor.
type LeitorDeCondicaoDeCompra interface {
	CondicaoDeCompra(ctx context.Context, itemCode, supplierCode int64, mask string) (CondicaoDeCompra, error)
}

// aplicarCondicaoDeCompra põe a linha que o MRP sugeriu (quantidade na unidade
// de ESTOQUE) na unidade em que o fornecedor vende e com o preço da tabela.
//
// Antes: a chapa (1 UN = 84,78 kg) saía no pedido como 376,7932 kg, e a linha
// ia com o preço que a tela mandasse — R$ 0,00 quando ninguém digitava —, que
// passava pela alçada de valores sem parar em ninguém.
//
// A quantidade convertida sobe para as casas da unidade de compra (4,44 chapas
// viram 5: não se compra 0,44 chapa). Sem conversão entre as unidades a linha
// fica na de estoque, como sempre foi. Preço informado pelo comprador vale por
// unidade da linha; sem ele, vale a tabela; sem nenhum, a aprovação é recusada.
func (uc *ApprovePurchaseSuggestionUseCase) aplicarCondicaoDeCompra(ctx context.Context, item *poentity.PurchaseOrderItem, supplierCode int64, precoInformado decimal.Decimal) error {
	var cond CondicaoDeCompra
	if uc.Condicoes != nil {
		var err error
		if cond, err = uc.Condicoes.CondicaoDeCompra(ctx, item.ItemCode, supplierCode, item.Mask); err != nil {
			return err
		}
	}
	rotulo := cond.Rotulo
	if rotulo == "" {
		rotulo = fmt.Sprintf("%d", item.ItemCode)
	}
	var conv ports.UOMConverter
	estoque := ""
	if uc.Unidade != nil {
		conv = uc.Unidade.Conversor
		if uc.Unidade.Unidades != nil {
			u, err := uc.Unidade.Unidades.UnidadeDeEstoqueDoItem(ctx, item.ItemCode)
			if err != nil {
				return err
			}
			estoque = strings.ToUpper(strings.TrimSpace(u))
		}
	}
	linha := estoque
	compra := strings.ToUpper(strings.TrimSpace(cond.Unidade))
	if compra != "" && compra != estoque && estoque != "" && conv != nil {
		q, achou, err := ports.ConvertQuantityWithMask(ctx, conv, item.ItemCode, item.Mask, item.RequestedQty, estoque, compra)
		if err != nil {
			return fmt.Errorf("convertendo a sugestão para a unidade de compra %s: %w", compra, err)
		}
		if achou && q.IsPositive() {
			if cond.Casas >= 0 {
				q = q.RoundCeil(cond.Casas)
			}
			item.RequestedQty = q
			item.PurchaseUOM = &compra
			linha = compra
		}
	} else if compra != "" && compra == estoque && cond.Casas >= 0 {
		// Compra na própria unidade de estoque: a quantidade também vai às
		// casas dela, para cima. Sem isso saía pedido de 0,4067 barra (UN) — a
		// necessidade do MRP vem com 4 casas e ninguém compra fração de barra.
		if q := item.RequestedQty.RoundCeil(cond.Casas); q.IsPositive() {
			item.RequestedQty = q
		}
	}

	preco := precoInformado
	if !preco.IsPositive() && len(cond.Precos) > 0 {
		var err error
		if preco, err = precoDaFaixa(ctx, conv, item, cond.Precos, estoque, linha); err != nil {
			return err
		}
	}
	if !preco.IsPositive() {
		return errorsuc.NewValidationError(fmt.Sprintf("o item %s não tem preço para este fornecedor: informe o preço unitário ou cadastre-o na tabela de preço de compra (VSUP0120) — pedido a R$ 0,00 passaria pela alçada sem aprovação", rotulo))
	}
	item.UnitPrice = preco
	item.TotalPrice = poentity.TotalDaLinha(item.RequestedQty, preco, decimal.Zero)
	return nil
}

// precoDaFaixa escolhe a linha da tabela pela quantidade do pedido e converte
// o preço para a unidade da linha do pedido.
//
// A faixa ("a partir de 10 UN") está na unidade da linha da TABELA: comparar
// com a quantidade de estoque (376 kg ≥ 10) punha 5 chapas no preço de quem
// compra 10. A quantidade do pedido é levada para a unidade de cada linha da
// tabela antes de comparar; linha da tabela em unidade sem conversão não serve
// (nem o preço dela daria para converter).
func precoDaFaixa(ctx context.Context, conv ports.UOMConverter, item *poentity.PurchaseOrderItem, precos []PrecoDeTabela, estoque, linha string) (decimal.Decimal, error) {
	var semConversao *PrecoDeTabela
	menorFaixa := ""
	for i := range precos {
		p := precos[i]
		de := strings.ToUpper(strings.TrimSpace(p.Unidade))
		if de == "" {
			de = estoque
		}
		qtd := item.RequestedQty
		if de != "" && linha != "" && de != linha {
			if conv == nil {
				semConversao = &precos[i]
				continue
			}
			q, achou, err := ports.ConvertQuantityWithMask(ctx, conv, item.ItemCode, item.Mask, item.RequestedQty, linha, de)
			if err != nil {
				return decimal.Zero, fmt.Errorf("levando a quantidade para a unidade da tabela %d (%s): %w", p.Tabela, de, err)
			}
			if !achou {
				semConversao = &precos[i]
				continue
			}
			qtd = q
		}
		if p.MinQtd.GreaterThan(qtd) {
			menorFaixa = fmt.Sprintf("a tabela %d só tem preço a partir de %s %s e o pedido é de %s %s", p.Tabela, p.MinQtd.String(), de, qtd.Round(4).String(), de)
			continue
		}
		if de == "" || linha == "" || de == linha {
			return p.Preco, nil
		}
		convertido, achou, err := conv.ConvertUnitPrice(ctx, item.ItemCode, p.Preco, de, linha)
		if err != nil {
			return decimal.Zero, fmt.Errorf("convertendo o preço da tabela para %s: %w", linha, err)
		}
		if !achou || !convertido.IsPositive() {
			semConversao = &precos[i]
			continue
		}
		return convertido.Round(6), nil
	}
	if semConversao != nil {
		de := strings.ToUpper(strings.TrimSpace(semConversao.Unidade))
		return decimal.Zero, errorsuc.NewValidationError(fmt.Sprintf("o preço da tabela %d está por %s e o item não tem conversão para %s: informe o preço por %s ou cadastre a conversão (VSUP0110)", semConversao.Tabela, de, linha, linha))
	}
	if menorFaixa != "" {
		return decimal.Zero, errorsuc.NewValidationError(menorFaixa + ": informe o preço unitário ou cadastre a faixa na tabela de preço de compra (VSUP0120)")
	}
	return decimal.Zero, nil
}
