package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type ListInventoriesUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *ListInventoriesUseCase) Execute(ctx context.Context) ([]*entity.PhysicalInventory, error) {
	if !uc.Auth.CanListInventories(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListInventories(ctx)
}

func (uc *ListInventoriesUseCase) ByStatus(ctx context.Context, status string) ([]*entity.PhysicalInventory, error) {
	if !uc.Auth.CanListInventories(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListInventoriesByStatus(ctx, status)
}
