package mrp_calculation_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	mrpservice "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service"
)

var ErrInvalidPlanCode = errors.New("plan_code must be greater than zero")
var ErrInvalidInitialOrderNumber = errors.New("initial_order_number must be greater than zero")

type RunMRPCalculationUseCase struct {
	Service      mrpservice.MRPService
	Auth         ports.AuthService
	AutoReleaser interface {
		ExecuteAutoRelease(context.Context, int64) error
	}
}

func (uc *RunMRPCalculationUseCase) Execute(ctx context.Context, dto request.RunMRPCalculationDTO) (*response.MRPCalculationLogResponse, error) {
	if dto.PlanCode <= 0 {
		return nil, ErrInvalidPlanCode
	}
	if dto.InitialOrderNumber <= 0 {
		return nil, ErrInvalidInitialOrderNumber
	}
	if !uc.Auth.CanRunMRPCalculation(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	log, err := uc.Service.Calculate(ctx, dto.PlanCode, dto.InitialOrderNumber, dto.GenerateLLC)
	if err != nil {
		return nil, err
	}
	if uc.AutoReleaser != nil {
		if err := uc.AutoReleaser.ExecuteAutoRelease(ctx, dto.PlanCode); err != nil {
			return nil, err
		}
	}
	return toMRPCalculationLogResponse(log), nil
}
