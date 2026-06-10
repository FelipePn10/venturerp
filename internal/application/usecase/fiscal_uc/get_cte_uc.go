package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type GetCTeUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *GetCTeUseCase) Execute(ctx context.Context, id int64) (*response.FiscalCTeResponse, error) {
	if !uc.Auth.CanGetFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	c, err := uc.Repo.GetCTeByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toFiscalCTeResponse(c), nil
}
