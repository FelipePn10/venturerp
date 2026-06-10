package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type ListFiscalEntriesUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *ListFiscalEntriesUseCase) Execute(ctx context.Context) ([]*response.FiscalEntryResponse, error) {
	if !uc.Auth.CanListFiscalEntries(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	list, err := uc.Repo.ListEntries(ctx)
	if err != nil {
		return nil, err
	}
	return toFiscalEntryResponses(list), nil
}
