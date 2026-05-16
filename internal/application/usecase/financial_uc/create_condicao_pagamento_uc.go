package financial_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type CreateCondicaoPagamentoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CreateCondicaoPagamentoUseCase) Execute(
	ctx context.Context, dto request.CreateCondicaoPagamentoDTO,
) (*entity.CondicaoPagamento, error) {
	if !uc.Auth.CanCreateCondicaoPagamento(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	c := &entity.CondicaoPagamento{
		Nome:     dto.Nome,
		Parcelas: []byte(dto.Parcelas),
		Ativo:    true,
	}

	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()

	created, err := uc.Repo.CreateCondicaoPagamento(ctx, c)
	if err != nil {
		return nil, err
	}
	return created, nil
}
