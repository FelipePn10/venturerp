package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type ListStockMovementsUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *ListStockMovementsUseCase) Execute(ctx context.Context) ([]*response.StockMovementResponse, error) {
	if !uc.Auth.CanListStockMovements(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListMovements(ctx)
	if err != nil {
		return nil, err
	}
	return toStockMovementResponses(list), nil
}

func (uc *ListStockMovementsUseCase) ByItem(ctx context.Context, itemCode int64) ([]*response.StockMovementResponse, error) {
	if !uc.Auth.CanListStockMovements(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListMovementsByItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toStockMovementResponses(list), nil
}

func (uc *ListStockMovementsUseCase) ByWarehouse(ctx context.Context, warehouseID int64) ([]*response.StockMovementResponse, error) {
	if !uc.Auth.CanListStockMovements(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListMovementsByWarehouse(ctx, warehouseID)
	if err != nil {
		return nil, err
	}
	return toStockMovementResponses(list), nil
}
