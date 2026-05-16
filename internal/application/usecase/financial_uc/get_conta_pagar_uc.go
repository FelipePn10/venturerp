package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type GetContaPagarUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetContaPagarUseCase) Execute(ctx context.Context, id int64) (*entity.ContaPagar, error) {
	if !uc.Auth.CanGetContaPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetContaPagar(ctx, id)
}
