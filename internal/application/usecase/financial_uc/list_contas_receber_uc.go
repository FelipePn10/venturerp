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

type ListContasReceberUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListContasReceberUseCase) Execute(
	ctx context.Context, dto request.ListContasReceberFilter,
) ([]*entity.ContaReceber, error) {
	if !uc.Auth.CanListContasReceber(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	filters := repository.CRFilter{
		Status:    dto.Status,
		ClienteID: dto.ClienteID,
	}
	if dto.StartDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.StartDate)
		filters.StartDate = &t
	}
	if dto.EndDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.EndDate)
		filters.EndDate = &t
	}

	return uc.Repo.ListContasReceber(ctx, filters)
}
