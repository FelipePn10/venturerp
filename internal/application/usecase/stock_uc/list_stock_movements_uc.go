package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type ListStockMovementsUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *ListStockMovementsUseCase) Execute(ctx context.Context) ([]*entity.StockMovement, error) {
	if !uc.Auth.CanListStockMovements(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListMovements(ctx)
}

func (uc *ListStockMovementsUseCase) ByItem(ctx context.Context, itemCode int64) ([]*entity.StockMovement, error) {
	if !uc.Auth.CanListStockMovements(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListMovementsByItem(ctx, itemCode)
}

func (uc *ListStockMovementsUseCase) ByWarehouse(ctx context.Context, warehouseID int64) ([]*entity.StockMovement, error) {
	if !uc.Auth.CanListStockMovements(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListMovementsByWarehouse(ctx, warehouseID)
}
