package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type CancelContaReceberUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CancelContaReceberUseCase) Execute(ctx context.Context, id int64) error {
	if !uc.Auth.CanCancelContaReceber(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.CancelContaReceber(ctx, id)
}
