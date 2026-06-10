package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type GetContaReceberUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetContaReceberUseCase) Execute(ctx context.Context, id int64) (*response.ContaReceberResponse, error) {
	if !uc.Auth.CanGetContaReceber(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	c, err := uc.Repo.GetContaReceber(ctx, id)
	if err != nil {
		return nil, err
	}
	return toContaReceberResponse(c), nil
}
