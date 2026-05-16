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
	return uc.Repo.CloseInventory(ctx, id)
}
