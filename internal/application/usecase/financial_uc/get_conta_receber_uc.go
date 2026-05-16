package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type GetContaReceberUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetContaReceberUseCase) Execute(ctx context.Context, id int64) (*entity.ContaReceber, error) {
	if !uc.Auth.CanGetContaReceber(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetContaReceber(ctx, id)
}
