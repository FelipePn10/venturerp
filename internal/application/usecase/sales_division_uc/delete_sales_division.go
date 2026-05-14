package sales_division_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/repository"
)

type DeleteSalesDivisionUseCase struct {
	Repo repository.SalesDivisionRepository
	Auth ports.AuthService
}

func (uc *DeleteSalesDivisionUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanDeleteSalesDivision(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.Delete(ctx, code)
}
