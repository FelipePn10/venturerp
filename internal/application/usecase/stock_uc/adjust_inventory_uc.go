package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type AdjustInventoryUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *AdjustInventoryUseCase) Execute(ctx context.Context, dto request.AdjustInventoryItemDTO) error {
	if !uc.Auth.CanAdjustInventory(ctx) {
		return errorsuc.ErrUnauthorized
	}

	item := &entity.PhysicalInventoryItem{
		InventoryID:      dto.InventoryID,
		ItemCode:         dto.ItemCode,
		Mask:             dto.Mask,
		WarehouseID:      dto.WarehouseID,
		AdjustmentType:   &dto.AdjustmentType,
		AdjustmentReason: dto.AdjustmentReason,
	}

	return uc.Repo.AdjustInventoryItem(ctx, item)
}
