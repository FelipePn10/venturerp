package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type CancelContaPagarUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CancelContaPagarUseCase) Execute(ctx context.Context, id int64) error {
	if !uc.Auth.CanCancelContaPagar(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.CancelContaPagar(ctx, id)
}
