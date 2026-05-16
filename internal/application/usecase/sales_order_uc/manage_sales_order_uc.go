package sales_order_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

type CancelSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *CancelSalesOrderUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Cancel(ctx, code)
}

type BlockSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *BlockSalesOrderUseCase) Execute(ctx context.Context, dto request.BlockSalesOrderDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Block(ctx, dto.Code, dto.Reason)
}

type UnblockSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *UnblockSalesOrderUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Unblock(ctx, code)
}

type ChangeStatusSalesOrderUseCase struct {
	Repo repository.SalesOrderRepository
	Auth ports.AuthService
}

func (uc *ChangeStatusSalesOrderUseCase) Execute(ctx context.Context, dto request.ChangeStatusDTO) error {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.ChangeStatus(ctx, dto.Code, entity.SalesOrderStatus(dto.Status))
}
