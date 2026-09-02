package sales_division_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_division/repository"
)

type SetSalesDivisionStatusUseCase struct {
	Repo repository.SalesDivisionRepository
	Auth ports.AuthService
}

func (uc *SetSalesDivisionStatusUseCase) Execute(ctx context.Context, code int64, active bool) (*response.SalesDivisionResponse, error) {
	if !uc.Auth.CanDeleteSalesDivision(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if code <= 0 {
		return nil, errorsuc.NewValidationError("código da divisão de vendas inválido")
	}
	division, err := uc.Repo.SetActive(ctx, code, active)
	if err != nil {
		return nil, err
	}
	return toSalesDivisionResponse(division), nil
}
