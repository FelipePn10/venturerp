package sales_division_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/repository"
)

type GetSalesDivisionUseCase struct {
	Repo repository.SalesDivisionRepository
	Auth ports.AuthService
}

func (uc *GetSalesDivisionUseCase) Execute(
	ctx context.Context,
	code int64,
) (*response.SalesDivisionResponse, error) {
	if !uc.Auth.CanGetSalesDivision(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	sd, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toSalesDivisionResponse(sd), nil
}
