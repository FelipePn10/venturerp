package mrp_calculation_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
)

type ListMRPExceptionsUseCase struct {
	Repo repository.MRPCalculationRepository
	Auth ports.AuthService
}

func (uc *ListMRPExceptionsUseCase) Execute(
	ctx context.Context,
	planCode int64,
) ([]*entity.MRPExceptionMessage, error) {
	if !uc.Auth.CanListMRPExceptions(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListExceptionsByPlan(ctx, planCode)
}
