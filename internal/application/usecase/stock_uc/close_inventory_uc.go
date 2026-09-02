package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type CloseInventoryUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *CloseInventoryUseCase) Execute(ctx context.Context, id int64) error {
	if !uc.Auth.CanCloseInventory(ctx) {
		return errorsuc.ErrUnauthorized
	}
	inv, err := uc.Repo.GetInventory(ctx, id)
	if err != nil {
		return err
	}
	if inv.Status != "OPEN" {
		return errorsuc.NewValidationError("somente inventário aberto pode ser fechado")
	}
	items, err := uc.Repo.ListInventoryItems(ctx, id)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return errorsuc.NewValidationError("inventário sem itens não pode ser fechado")
	}
	for _, item := range items {
		if item.CountedQty == nil {
			return errorsuc.NewValidationError("todos os itens devem ser contados antes do fechamento")
		}
		if item.DifferenceQty != nil && *item.DifferenceQty != 0 && !item.IsAdjusted {
			return errorsuc.NewValidationError("todas as divergências devem ser ajustadas antes do fechamento")
		}
	}
	return uc.Repo.CloseInventory(ctx, id)
}
