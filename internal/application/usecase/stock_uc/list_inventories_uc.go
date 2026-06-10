package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type ListInventoriesUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *ListInventoriesUseCase) Execute(ctx context.Context) ([]*response.PhysicalInventoryResponse, error) {
	if !uc.Auth.CanListInventories(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListInventories(ctx)
	if err != nil {
		return nil, err
	}
	return toPhysicalInventoryResponses(list), nil
}

func (uc *ListInventoriesUseCase) ByStatus(ctx context.Context, status string) ([]*response.PhysicalInventoryResponse, error) {
	if !uc.Auth.CanListInventories(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListInventoriesByStatus(ctx, status)
	if err != nil {
		return nil, err
	}
	return toPhysicalInventoryResponses(list), nil
}
