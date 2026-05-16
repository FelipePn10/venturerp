package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type ListFiscalExitsUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *ListFiscalExitsUseCase) Execute(ctx context.Context) ([]*entity.FiscalExit, error) {
	if !uc.Auth.CanListFiscalExits(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.ListExits(ctx)
}
