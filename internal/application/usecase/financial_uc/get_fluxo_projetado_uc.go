package financial_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type GetFluxoProjetadoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *GetFluxoProjetadoUseCase) Execute(ctx context.Context, startDate string) ([]*repository.ProjectedFlow, error) {
	if !uc.Auth.CanGetFluxoProjetado(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	start, _ := time.Parse("2006-01-02", startDate)

	return uc.Repo.GetFluxoProjetado(ctx, start)
}
