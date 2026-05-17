package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type ListCTeUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *ListCTeUseCase) Execute(ctx context.Context) ([]*entity.FiscalCTe, error) {
	if !uc.Auth.CanListFiscalEntries(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListCTe(ctx)
}
