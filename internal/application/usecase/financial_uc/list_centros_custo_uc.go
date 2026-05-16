package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type ListCentrosCustoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListCentrosCustoUseCase) Execute(ctx context.Context) ([]*entity.CentroCusto, error) {
	if !uc.Auth.CanListCentrosCustoFinancial(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListCentrosCusto(ctx)
}
