package purchase_order_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

type CancelPurchaseOrderUseCase struct {
	Repo repository.PurchaseOrderRepository
	Auth ports.AuthService
}

// Execute cancela o pedido inteiro. Pedido com material já recebido não se
// cancela — o recebido existe no estoque e na nota; o caminho é eliminar o
// saldo das linhas que faltam.
func (uc *CancelPurchaseOrderUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	po, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return err
	}
	if !po.IsActive || po.Status == entity.PurchaseOrderStatusCANCELLED {
		return errorsuc.NewValidationError(fmt.Sprintf("o pedido %d já está cancelado", code))
	}
	itens, err := uc.Repo.ListItems(ctx, code)
	if err != nil {
		return err
	}
	for _, it := range itens {
		if it.ReceivedQty > 0 {
			return errorsuc.NewValidationError(fmt.Sprintf("o pedido %d já tem recebimento (linha %d); elimine o saldo das linhas em aberto em vez de cancelar o pedido", code, it.Sequence))
		}
	}
	return uc.Repo.Cancel(ctx, code)
}
