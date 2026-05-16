package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type GetStockBalanceUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *GetStockBalanceUseCase) Execute(ctx context.Context, itemCode int64, mask string, warehouseID int64) (*entity.StockBalance, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetBalance(ctx, itemCode, mask, warehouseID)
}

func (uc *GetStockBalanceUseCase) List(ctx context.Context) ([]*entity.StockBalance, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListBalances(ctx)
}

func (uc *GetStockBalanceUseCase) ByWarehouse(ctx context.Context, warehouseID int64) ([]*entity.StockBalance, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListBalancesByWarehouse(ctx, warehouseID)
}

func (uc *GetStockBalanceUseCase) ByItem(ctx context.Context, itemCode int64) ([]*entity.StockBalance, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListBalancesByItem(ctx, itemCode)
}
