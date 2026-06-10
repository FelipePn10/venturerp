package sales_division_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/repository"
)

type ListSalesDivisionsUseCase struct {
	Repo repository.SalesDivisionRepository
	Auth ports.AuthService
}

func (uc *ListSalesDivisionsUseCase) Execute(
	ctx context.Context,
) ([]*response.SalesDivisionResponse, error) {
	if !uc.Auth.CanListSalesDivisions(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	divisions, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return toSalesDivisionResponses(divisions), nil
}
