package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type CountInventoryItemUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *CountInventoryItemUseCase) Execute(ctx context.Context, dto request.CountInventoryItemDTO) error {
	if !uc.Auth.CanCountInventoryItem(ctx) {
		return errorsuc.ErrUnauthorized
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}
	dto.CountedBy = actor
	inv, err := uc.Repo.GetInventory(ctx, dto.InventoryID)
	if err != nil {
		return err
	}
	if inv.Status != "OPEN" {
		return errorsuc.NewValidationError("somente inventário aberto pode receber contagem")
	}
	if dto.ItemCode <= 0 || dto.WarehouseID != inv.WarehouseID || dto.CountedQty < 0 {
		return errorsuc.NewValidationError("item, almoxarifado do inventário e quantidade contada válida são obrigatórios")
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

	item.CountedBy = &dto.CountedBy

	return uc.Repo.CountInventoryItem(ctx, item)
}
