package service

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/shopspring/decimal"
)

// FonteDeCompra: ver entity.FonteDeCompra.
type FonteDeCompra = entity.FonteDeCompra

// FontesDeCompraReader lê as fontes válidas do item, já ordenadas por
// preferencial e ranking. Opcional no repositório do MRP.
type FontesDeCompraReader interface {
	FontesDeCompra(ctx context.Context, itemCode int64) ([]FonteDeCompra, error)
}

// escolherFonte decide de quem comprar esta necessidade.
//
// Com cota (algum fornecedor da máscara com %): o índice de cota do SAP —
// vai INTEIRA para o fornecedor de menor (alocado ÷ cota); empate, o de melhor
// ranking. Ao longo do tempo as compras se repartem pelas cotas sem picotar
// cada pedido em pedaços abaixo do mínimo de cada um.
//
// Sem cota: o preferencial; sem ele, o de melhor ranking (a ordem já vem assim).
// Devolve o índice em fontes, ou -1 se não há fonte para a máscara.
func escolherFonte(fontes []FonteDeCompra, mask string) int {
	escolhida := -1
	var melhorIndice decimal.Decimal
	comCota := false
	for i := range fontes {
		f := &fontes[i]
		if f.Mask != "" && f.Mask != mask {
			continue
		}
		if f.QuotaPercent == nil || !f.QuotaPercent.IsPositive() {
			continue
		}
		indice := f.Alocado.Div(*f.QuotaPercent)
		if !comCota || indice.LessThan(melhorIndice) ||
			(indice.Equal(melhorIndice) && f.Ranking < fontes[escolhida].Ranking) {
			escolhida, melhorIndice, comCota = i, indice, true
		}
	}
	if comCota {
		return escolhida
	}
	for i := range fontes {
		if fontes[i].Mask == "" || fontes[i].Mask == mask {
			return i
		}
	}
	return -1
}

// prazoDaFonte: o prazo da compra é o do fornecedor ESCOLHIDO; sem prazo
// próprio, o prazo de compra do item — não o do preferencial, que é o que o
// prazo do cadastro devolveria quando a cota escolhe outro fornecedor.
// (Prazo de cenário ou de regra da máscara vence e é tratado antes.)
func prazoDaFonte(fonte *FonteDeCompra, prazoCompraDoItem *int) int {
	if fonte.LeadTimeDays > 0 {
		return fonte.LeadTimeDays
	}
	return prazoInformado(prazoCompraDoItem)
}
