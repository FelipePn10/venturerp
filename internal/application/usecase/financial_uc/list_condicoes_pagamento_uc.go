package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type ListCondicoesPagamentoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListCondicoesPagamentoUseCase) Execute(ctx context.Context) ([]*response.CondicaoPagamentoResponse, error) {
	if !uc.Auth.CanListCondicoesPagamento(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListCondicoesPagamento(ctx)
	if err != nil {
		return nil, err
	}
	return toCondicaoPagamentoResponses(list), nil
}
