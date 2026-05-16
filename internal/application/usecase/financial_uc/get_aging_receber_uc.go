package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type GetAgingReceberUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetAgingReceberUseCase) Execute(ctx context.Context) ([]*repository.AgingResult, error) {
	if !uc.Auth.CanGetAgingReceber(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetAgingContasReceber(ctx)
}
