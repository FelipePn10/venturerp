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

type CreateCentroCustoUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CreateCentroCustoUseCase) Execute(
	ctx context.Context, dto request.CreateCentroCustoDTO,
) (*entity.CentroCusto, error) {
	if !uc.Auth.CanCreateCentroCustoFinancial(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	c := &entity.CentroCusto{
		Codigo:    dto.Codigo,
		Descricao: dto.Descricao,
		Tipo:      dto.Tipo,
		IsActive:  true,
	}

	c.CreatedAt = time.Now()

	created, err := uc.Repo.CreateCentroCusto(ctx, c)
	if err != nil {
		return nil, err
	}
	return created, nil
}
