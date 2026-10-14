package financial_uc

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type GetFluxoCaixaUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetFluxoCaixaUseCase) Execute(ctx context.Context, startDate, endDate string) ([]*response.FluxoCaixaResponse, error) {
	if !uc.Auth.CanGetFluxoCaixa(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	// Período em branco segue como antes (sem limite daquele lado); data mal
	// digitada é recusada — antes virava 0001-01-01 e o fluxo vinha errado.
	var start, end time.Time
	if d, msg := datetime.Opcional("data inicial", &startDate); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	} else if d != nil {
		start = *d
	}
	if d, msg := datetime.Opcional("data final", &endDate); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	} else if d != nil {
		end = *d
	}

	list, err := uc.Repo.GetFluxoCaixa(ctx, start, end)
	if err != nil {
		return nil, err
	}
	return toFluxoCaixaResponses(list), nil
}
