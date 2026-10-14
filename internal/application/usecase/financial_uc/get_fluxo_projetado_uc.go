package financial_uc

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
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

	var start time.Time
	if d, msg := datetime.Opcional("data inicial", &startDate); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	} else if d != nil {
		start = *d
	}

	return uc.Repo.GetFluxoProjetado(ctx, start)
}
