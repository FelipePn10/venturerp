package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type ListContasBancariasUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ListContasBancariasUseCase) Execute(ctx context.Context) ([]*entity.ContaBancaria, error) {
	if !uc.Auth.CanListContasBancarias(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListContasBancarias(ctx)
}
