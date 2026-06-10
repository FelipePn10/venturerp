package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type GetFiscalConfigUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *GetFiscalConfigUseCase) Execute(ctx context.Context) (*response.FiscalConfigResponse, error) {
	if !uc.Auth.CanManageFiscalConfig(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	c, err := uc.Repo.GetFiscalConfig(ctx)
	if err != nil {
		return nil, err
	}
	return toFiscalConfigResponse(c), nil
}
