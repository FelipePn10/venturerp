package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type CancelFiscalExitUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *CancelFiscalExitUseCase) Execute(ctx context.Context, id int64) (*entity.FiscalExit, error) {
	if !uc.Auth.CanCancelFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.UpdateExitStatus(ctx, id, entity.ExitStatusCancelled)
}
