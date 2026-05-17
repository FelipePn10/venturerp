package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type GetCTeUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *GetCTeUseCase) Execute(ctx context.Context, id int64) (*entity.FiscalCTe, error) {
	if !uc.Auth.CanGetFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetCTeByID(ctx, id)
}
