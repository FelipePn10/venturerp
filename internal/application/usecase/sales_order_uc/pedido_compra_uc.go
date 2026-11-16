package sales_order_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
)

type pedidoCompraRepo interface {
	SetCustomerPO(ctx context.Context, code int64, po *string) (int64, error)
}

// SetCustomerPOUseCase informa ou corrige o pedido de compra do cliente num
// pedido ainda não faturado (quem esqueceu na criação não fica travado na
// confirmação do cliente que exige o número).
type SetCustomerPOUseCase struct {
	Repo pedidoCompraRepo
	Auth ports.AuthService
}

func (uc *SetCustomerPOUseCase) Execute(ctx context.Context, code int64, po *string) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	if err := validarPedidoDeCompra(po); err != nil {
		return err
	}
	n, err := uc.Repo.SetCustomerPO(ctx, code, po)
	if err != nil {
		return err
	}
	if n == 0 {
		return errorsuc.NewValidationError(fmt.Sprintf("pedido %d não encontrado ou já faturado/cancelado", code))
	}
	return nil
}
