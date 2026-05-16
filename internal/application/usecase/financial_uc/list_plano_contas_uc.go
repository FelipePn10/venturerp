package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type ListPlanoContasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListPlanoContasUseCase) Execute(ctx context.Context) ([]*entity.PlanoContas, error) {
	if !uc.Auth.CanListPlanoContas(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListPlanoContas(ctx)
}
