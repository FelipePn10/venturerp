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

type CreatePlanoContasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CreatePlanoContasUseCase) Execute(
	ctx context.Context, dto request.CreatePlanoContasDTO,
) (*entity.PlanoContas, error) {
	if !uc.Auth.CanCreatePlanoContas(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	p := &entity.PlanoContas{
		Codigo:     dto.Codigo,
		Descricao:  dto.Descricao,
		Tipo:       dto.Tipo,
		Natureza:   dto.Natureza,
		ParentCode: dto.ParentCode,
		Nivel:      dto.Nivel,
		IsActive:   true,
	}

	p.CreatedAt = time.Now()

	created, err := uc.Repo.CreatePlanoContas(ctx, p)
	if err != nil {
		return nil, err
	}
	return created, nil
}
