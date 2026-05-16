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

type ListContasPagarUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListContasPagarUseCase) Execute(
	ctx context.Context, dto request.ListContasPagarFilter,
) ([]*entity.ContaPagar, error) {
	if !uc.Auth.CanListContasPagar(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	filters := repository.CPFilter{
		Status:      dto.Status,
		FornecedorID: dto.FornecedorID,
	}
	if dto.StartDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.StartDate)
		filters.StartDate = &t
	}
	if dto.EndDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.EndDate)
		filters.EndDate = &t
	}

	return uc.Repo.ListContasPagar(ctx, filters)
}
