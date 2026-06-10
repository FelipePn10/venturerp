package fiscal_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/repository"
)

type GetFiscalEntryUseCase struct {
	Repo repository.FiscalRepository
	Auth ports.AuthService
}

func (uc *GetFiscalEntryUseCase) Execute(ctx context.Context, id int64) (*response.FiscalEntryResponse, error) {
	if !uc.Auth.CanGetFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	entry, err := uc.Repo.GetEntryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	items, _ := uc.Repo.GetEntryItems(ctx, id)
	entry.Itens = items

	return toFiscalEntryResponse(entry), nil
}
