package purchase_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

type GetPurchaseOrderUseCase struct {
	Repo repository.PurchaseOrderRepository
	Auth ports.AuthService
}

func (uc *GetPurchaseOrderUseCase) Execute(ctx context.Context, code int64) (*entity.PurchaseOrder, error) {
	if !uc.Auth.CanGetPurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	o, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	items, err := uc.Repo.ListItems(ctx, code)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return o, nil
}

type ListPurchaseOrdersUseCase struct {
	Repo repository.PurchaseOrderRepository
	Auth ports.AuthService
}

func (uc *ListPurchaseOrdersUseCase) Execute(ctx context.Context) ([]*entity.PurchaseOrder, error) {
	if !uc.Auth.CanListPurchaseOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.List(ctx)
}

type ListPurchaseOrdersBySupplierUseCase struct {
	Repo repository.PurchaseOrderRepository
	Auth ports.AuthService
}

func (uc *ListPurchaseOrdersBySupplierUseCase) Execute(ctx context.Context, supplierCode int64) ([]*entity.PurchaseOrder, error) {
	if !uc.Auth.CanListPurchaseOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListBySupplier(ctx, supplierCode)
}

type ListPurchaseOrdersByStatusUseCase struct {
	Repo repository.PurchaseOrderRepository
	Auth ports.AuthService
}

func (uc *ListPurchaseOrdersByStatusUseCase) Execute(ctx context.Context, status string) ([]*entity.PurchaseOrder, error) {
	if !uc.Auth.CanListPurchaseOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListByStatus(ctx, entity.PurchaseOrderStatus(status))
}
