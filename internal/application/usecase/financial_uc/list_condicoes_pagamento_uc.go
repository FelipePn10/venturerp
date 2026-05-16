package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type ListCondicoesPagamentoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListCondicoesPagamentoUseCase) Execute(ctx context.Context) ([]*entity.CondicaoPagamento, error) {
	if !uc.Auth.CanListCondicoesPagamento(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListCondicoesPagamento(ctx)
}
