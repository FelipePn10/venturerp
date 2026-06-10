package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type ListPlanoContasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListPlanoContasUseCase) Execute(ctx context.Context) ([]*response.PlanoContasResponse, error) {
	if !uc.Auth.CanListPlanoContas(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListPlanoContas(ctx)
	if err != nil {
		return nil, err
	}
	return toPlanoContasResponses(list), nil
}
