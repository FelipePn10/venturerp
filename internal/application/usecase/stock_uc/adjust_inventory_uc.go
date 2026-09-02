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
	inv, err := uc.Repo.GetInventory(ctx, dto.InventoryID)
	if err != nil {
		return err
	}
	if inv.Status != "OPEN" {
		return errorsuc.NewValidationError("somente inventário aberto pode ser ajustado")
	}
	if dto.ItemCode <= 0 || dto.WarehouseID != inv.WarehouseID || (dto.AdjustmentType != "IN" && dto.AdjustmentType != "OUT" && dto.AdjustmentType != "NONE") {
		return errorsuc.NewValidationError("item, almoxarifado e tipo de ajuste IN, OUT ou NONE são obrigatórios")
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
