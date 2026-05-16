package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type GetFiscalExitUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *GetFiscalExitUseCase) Execute(ctx context.Context, id int64) (*entity.FiscalExit, error) {
	if !uc.Auth.CanGetFiscalExit(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	exit, err := uc.Repo.GetExitByID(ctx, id)
	if err != nil {
		return nil, err
	}

	items, _ := uc.Repo.GetExitItems(ctx, id)
	exit.Itens = items

	return exit, nil
}
