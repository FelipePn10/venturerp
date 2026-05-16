package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/google/uuid"
)

type CountInventoryItemUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *CountInventoryItemUseCase) Execute(ctx context.Context, dto request.CountInventoryItemDTO) error {
	if !uc.Auth.CanCountInventoryItem(ctx) {
		return errorsuc.ErrUnauthorized
	}

	item := &entity.PhysicalInventoryItem{
		InventoryID:      dto.InventoryID,
		ItemCode:         dto.ItemCode,
		Mask:             dto.Mask,
		WarehouseID:      dto.WarehouseID,
		CountedQty:       &dto.CountedQty,
		UnitCost:         dto.UnitCost,
		AdjustmentType:   dto.AdjustmentType,
		AdjustmentReason: dto.AdjustmentReason,
	}

	if dto.CountedBy != nil {
		id, err := uuid.Parse(*dto.CountedBy)
		if err == nil {
			item.CountedBy = &id
		}
	}

	return uc.Repo.CountInventoryItem(ctx, item)
}
