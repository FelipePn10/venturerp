package mrp_calculation_uc

import (
	"context"
	"errors"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	mrpservice "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/service"
	planrepo "github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository"
)

var ErrInvalidPlanCode = errorsuc.NewValidationError("o campo 'plan_code' deve ser maior que zero")
var ErrInvalidInitialOrderNumber = errorsuc.NewValidationError("o campo 'initial_order_number' deve ser maior que zero")

type RunMRPCalculationUseCase struct {
	Service      mrpservice.MRPService
	Auth         ports.AuthService
	Plans        planrepo.ProductionPlanRepository
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
	if uc.Plans == nil {
		return nil, errors.New("repositório de planos não configurado")
	}
	plan, err := uc.Plans.GetByCode(ctx, dto.PlanCode)
	if err != nil {
		if errors.Is(err, planrepo.ErrNotFound) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("plano de referência %d não encontrado — selecione um cadastro existente", dto.PlanCode))
		}
		return nil, err
	}
	if plan.IsScenario {
		return nil, errorsuc.NewValidationError("este plano pertence a um cenário de simulação; calcule pelo próprio cenário")
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
