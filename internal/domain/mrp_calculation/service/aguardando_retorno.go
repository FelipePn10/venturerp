package service

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/shopspring/decimal"
)

// producaoAguardandoRetornoReader lê, por item, o que a OF de beneficiamento já
// entregou e a nota de retorno ainda não levou. Opcional no repositório.
type producaoAguardandoRetornoReader interface {
	ProducaoAguardandoRetorno(ctx context.Context) (map[int64]decimal.Decimal, error)
}

// somarAguardandoRetorno trata a peça pronta do cliente como estoque para o
// cálculo. A OF de beneficiamento não dá entrada no estoque próprio e o pedido só
// é atendido quando a nota é autorizada: sem isto, a linha do pedido seguiria
// aberta sem nada que a cobrisse, e o MRP mandaria produzir de novo o que já está
// pronto esperando a nota.
func somarAguardandoRetorno(aguardando map[int64]decimal.Decimal, snapshots map[int64]*entity.StockSnapshot) {
	for item, quantidade := range aguardando {
		if !quantidade.IsPositive() {
			continue
		}
		copia := entity.StockSnapshot{ItemCode: item}
		if atual := snapshots[item]; atual != nil {
			copia = *atual
		}
		copia.Quantity = copia.Quantity.Add(quantidade)
		snapshots[item] = &copia
	}
}
