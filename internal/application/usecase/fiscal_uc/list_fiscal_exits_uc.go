package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type ListFiscalExitsUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *ListFiscalExitsUseCase) Execute(ctx context.Context) ([]*response.FiscalExitResponse, error) {
	if !uc.Auth.CanListFiscalExits(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	list, err := uc.Repo.ListExits(ctx)
	if err != nil {
		return nil, err
	}
	return toFiscalExitResponses(list), nil
}
