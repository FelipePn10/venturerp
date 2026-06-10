package mrp_calculation_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
)

type ListMRPExceptionsUseCase struct {
	Repo repository.MRPCalculationRepository
	Auth ports.AuthService
}

func (uc *ListMRPExceptionsUseCase) Execute(
	ctx context.Context,
	planCode int64,
) ([]*response.MRPExceptionMessageResponse, error) {
	if !uc.Auth.CanListMRPExceptions(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListExceptionsByPlan(ctx, planCode)
	if err != nil {
		return nil, err
	}
	return toMRPExceptionResponses(list), nil
}
