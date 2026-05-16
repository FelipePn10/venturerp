package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type ApproveContaPagarUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ApproveContaPagarUseCase) Execute(ctx context.Context, id int64) error {
	if !uc.Auth.CanApproveContaPagar(ctx) {
		return errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}

	return uc.Repo.ApproveContaPagar(ctx, id, userID)
}
