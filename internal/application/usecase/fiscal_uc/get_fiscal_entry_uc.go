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
	// Docs, quando presente, devolve a nota completa: itens com o cadastro
	// conciliado, parcelas, totais por plano de contas e pendências.
	Docs        repository.FiscalEntryDocumentRepository
	Tolerancias ports.PurchaseToleranceEvaluator
	Auth        ports.AuthService
}

func (uc *GetFiscalEntryUseCase) Execute(ctx context.Context, id int64) (*response.FiscalEntryResponse, error) {
	if !uc.Auth.CanGetFiscalEntry(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	if uc.Docs != nil {
		doc, err := uc.Docs.GetEntryDocument(ctx, id)
		if err != nil {
			return nil, err
		}
		servico := &EntradaServico{Docs: uc.Docs, Fiscal: uc.Repo, Tolerancias: uc.Tolerancias}
		return servico.Responder(ctx, doc)
	}

	entry, err := uc.Repo.GetEntryByID(ctx, id)
	if err != nil {
		return nil, err
	}

	items, err := uc.Repo.GetEntryItems(ctx, id)
	if err != nil {
		return nil, err
	}
	entry.Itens = items

	return toFiscalEntryResponse(entry), nil
}
