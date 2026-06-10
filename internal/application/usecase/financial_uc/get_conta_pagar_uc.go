package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type GetContaPagarUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetContaPagarUseCase) Execute(ctx context.Context, id int64) (*response.ContaPagarResponse, error) {
	if !uc.Auth.CanGetContaPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	c, err := uc.Repo.GetContaPagar(ctx, id)
	if err != nil {
		return nil, err
	}
	return toContaPagarResponse(c), nil
}
