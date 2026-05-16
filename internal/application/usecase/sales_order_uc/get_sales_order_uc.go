package sales_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

type GetSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *GetSalesOrderUseCase) Execute(ctx context.Context, code int64) (*entity.SalesOrder, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
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

type ListSalesOrdersUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ListSalesOrdersUseCase) Execute(ctx context.Context) ([]*entity.SalesOrder, error) {
	if !uc.Auth.CanListSalesOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.List(ctx)
}

type ListSalesOrdersByCustomerUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ListSalesOrdersByCustomerUseCase) Execute(ctx context.Context, customerCode int64) ([]*entity.SalesOrder, error) {
	if !uc.Auth.CanListSalesOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListByCustomer(ctx, customerCode)
}

type ListSalesOrdersByStatusUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ListSalesOrdersByStatusUseCase) Execute(ctx context.Context, status string) ([]*entity.SalesOrder, error) {
	if !uc.Auth.CanListSalesOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListByStatus(ctx, entity.SalesOrderStatus(status))
}
