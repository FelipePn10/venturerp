package financial_uc

import (
	"context"
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

	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)

	list, err := uc.Repo.GetFluxoCaixa(ctx, start, end)
	if err != nil {
		return nil, err
	}
	return toFluxoCaixaResponses(list), nil
}
