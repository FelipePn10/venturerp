package sales_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

type GetSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *GetSalesOrderUseCase) Execute(ctx context.Context, code int64) (*response.SalesOrderResponse, error) {
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
	return toSalesOrderResponse(o), nil
}

type ListSalesOrdersUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ListSalesOrdersUseCase) Execute(ctx context.Context) ([]*response.SalesOrderResponse, error) {
	if !uc.Auth.CanListSalesOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	orders, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return toSalesOrderResponses(orders), nil
}

type ListSalesOrdersByCustomerUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ListSalesOrdersByCustomerUseCase) Execute(ctx context.Context, customerCode int64) ([]*response.SalesOrderResponse, error) {
	if !uc.Auth.CanListSalesOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	orders, err := uc.Repo.ListByCustomer(ctx, customerCode)
	if err != nil {
		return nil, err
	}
	return toSalesOrderResponses(orders), nil
}

type ListSalesOrdersByStatusUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ListSalesOrdersByStatusUseCase) Execute(ctx context.Context, status string) ([]*response.SalesOrderResponse, error) {
	if !uc.Auth.CanListSalesOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	orders, err := uc.Repo.ListByStatus(ctx, entity.SalesOrderStatus(status))
	if err != nil {
		return nil, err
	}
	return toSalesOrderResponses(orders), nil
}

type ListSalesOrdersAdvancedUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ListSalesOrdersAdvancedUseCase) Execute(ctx context.Context, filter repository.SalesOrderFilter) ([]*response.SalesOrderResponse, error) {
	if !uc.Auth.CanListSalesOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	orders, err := uc.Repo.ListAdvanced(ctx, filter)
	if err != nil {
		return nil, err
	}
	return toSalesOrderResponses(orders), nil
}

type SalesOrderReportUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *SalesOrderReportUseCase) Execute(ctx context.Context, filter repository.SalesOrderFilter) (*response.SalesOrderReportResponse, error) {
	if !uc.Auth.CanListSalesOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	report, err := uc.Repo.Report(ctx, filter)
	if err != nil {
		return nil, err
	}
	return &response.SalesOrderReportResponse{
		TotalOrders:            report.TotalOrders,
		TotalGross:             report.TotalGross,
		TotalNet:               report.TotalNet,
		OpenCount:              report.OpenCount,
		ConfirmedCount:         report.ConfirmedCount,
		InvoicedCount:          report.InvoicedCount,
		CancelledCount:         report.CancelledCount,
		BlockedCount:           report.BlockedCount,
		CommercialPendingCount: report.CommercialPendingCount,
		FinancialPendingCount:  report.FinancialPendingCount,
		ConferencePendingCount: report.ConferencePendingCount,
		DelayedCount:           report.DelayedCount,
	}, nil
}
