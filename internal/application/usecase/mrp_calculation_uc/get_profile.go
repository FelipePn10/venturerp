package mrp_calculation_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
)

type GetItemProfileUseCase struct {
	Repo repository.MRPCalculationRepository
	Auth ports.AuthService
}

func (uc *GetItemProfileUseCase) Execute(
	ctx context.Context,
	itemCode, planCode int64,
) ([]*response.MRPItemProfileResponse, error) {
	if !uc.Auth.CanRunMRPCalculation(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.GetProfiles(ctx, itemCode, planCode)
	if err != nil {
		return nil, err
	}
	return toMRPItemProfileResponses(list), nil
}
