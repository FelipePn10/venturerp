package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type GetFiscalConfigUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *GetFiscalConfigUseCase) Execute(ctx context.Context) (*entity.FiscalConfig, error) {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	return uc.Repo.GetFiscalConfig(ctx)
}
