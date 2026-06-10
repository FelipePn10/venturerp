package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type GetInventoryUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *GetInventoryUseCase) Execute(ctx context.Context, id int64) (*response.PhysicalInventoryResponse, error) {
	if !uc.Auth.CanGetInventory(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	inv, err := uc.Repo.GetInventory(ctx, id)
	if err != nil {
		return nil, err
	}
	items, err := uc.Repo.ListInventoryItems(ctx, inv.ID)
	if err != nil {
		return nil, err
	}
	inv.TotalItems = len(items)
	return toPhysicalInventoryResponse(inv), nil
}

func (uc *GetInventoryUseCase) ListItems(ctx context.Context, inventoryID int64) ([]*response.PhysicalInventoryItemResponse, error) {
	if !uc.Auth.CanGetInventory(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	items, err := uc.Repo.ListInventoryItems(ctx, inventoryID)
	if err != nil {
		return nil, err
	}
	return toPhysicalInventoryItemResponses(items), nil
}
