package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type ListCTeUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *ListCTeUseCase) Execute(ctx context.Context) ([]*response.FiscalCTeResponse, error) {
	if !uc.Auth.CanListFiscalEntries(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListCTe(ctx)
	if err != nil {
		return nil, err
	}
	return toFiscalCTeResponses(list), nil
}
