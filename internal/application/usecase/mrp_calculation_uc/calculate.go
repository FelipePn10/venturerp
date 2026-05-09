package mrp_calculation_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrpservice "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service"
)

type RunMRPCalculationUseCase struct {
	Service mrpservice.MRPService
	Auth    ports.AuthService
}

func (uc *RunMRPCalculationUseCase) Execute(ctx context.Context, dto request.RunMRPCalculationDTO) (*entity.MRPCalculationLog, error) {
	if !uc.Auth.CanRunMRPCalculation(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Service.Calculate(ctx, dto.PlanCode, dto.GenerateLLC)
}
