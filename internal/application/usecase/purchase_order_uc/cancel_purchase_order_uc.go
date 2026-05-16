package purchase_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

type CancelPurchaseOrderUseCase struct {
	Repo repository.PurchaseOrderRepository
	Auth ports.AuthService
}

func (uc *CancelPurchaseOrderUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Cancel(ctx, code)
}
