package production_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
)

type cadeiaRepository interface {
	CadeiaDaOF(ctx context.Context, orderID int64) (entity.CadeiaDaOF, error)
}

// Chain devolve a cadeia de fabricação da OF: os pedidos de venda que ela
// atende, as ordens pai que ela abastece e as ordens filhas dos componentes
// fabricados para ela (ver .ai/tasks/cadeia-of-pai-filha.md).
func (uc *ProductionMaterialControlUseCase) Chain(ctx context.Context, orderID int64) (*response.ProductionChainResponse, error) {
	if !uc.Auth.CanListOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	repo, ok := uc.Repo.(cadeiaRepository)
	if !ok {
		return nil, errorsuc.NewValidationError("a consulta da cadeia de fabricação não está disponível")
	}
	// OF inexistente ou de outra empresa é "não encontrada", não cadeia vazia.
	if _, err := uc.Repo.GetByCode(ctx, orderID); err != nil {
		return nil, err
	}
	cadeia, err := repo.CadeiaDaOF(ctx, orderID)
	if err != nil {
		return nil, err
	}
	out := &response.ProductionChainResponse{
		ProductionOrderID: orderID,
		SalesOrders:       make([]response.ProductionChainSalesOrder, 0, len(cadeia.Pedidos)),
		Parents:           elosDaCadeia(cadeia.Pais),
		Children:          elosDaCadeia(cadeia.Filhas),
	}
	for _, p := range cadeia.Pedidos {
		out.SalesOrders = append(out.SalesOrders, response.ProductionChainSalesOrder{
			SalesOrderCode: p.SalesOrderCode, SalesOrderNumber: p.SalesOrderNumber,
			SalesOrderItemCode: p.SalesOrderItemCode, Sequence: p.Sequence, Customer: p.Customer,
		})
	}
	return out, nil
}

func elosDaCadeia(elos []entity.OrdemDaCadeia) []response.ProductionChainOrder {
	out := make([]response.ProductionChainOrder, 0, len(elos))
	for _, o := range elos {
		out = append(out, response.ProductionChainOrder{
			PlannedOrderCode: o.PlannedOrderCode, ProductionOrderID: o.ProductionOrderID, OrderNumber: o.OrderNumber,
			ItemCode: o.ItemCode, BusinessCode: o.BusinessCode, Description: o.Description, Status: o.Status,
			Quantity: o.Quantity, Produced: o.Produced, Reserved: o.Reserved,
		})
	}
	return out
}
