package production_plan_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository"
)

type UpdateProductionPlanUseCase struct {
	Repo repository.ProductionPlanRepository
	Auth ports.AuthService
}

func (uc *UpdateProductionPlanUseCase) Execute(
	ctx context.Context,
	dto request.UpdateProductionPlanDTO,
) (*response.ProductionPlanResponse, error) {
	if !uc.Auth.CanUpdateProductionPlan(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	plan, err := uc.Repo.GetByCode(ctx, dto.Code)
	if err != nil {
		return nil, err
	}
	plan.Name = dto.Name
	plan.IndependentDemands = dto.IndependentDemands
	plan.GroupSameDateOrders = dto.GroupSameDateOrders
	plan.PlanningTypes = dto.PlanningTypes
	if err := plan.Configure(dto.Classification, dto.ClassItemCodes, dto.OrderItemCode, dto.Parameters); err != nil {
		return nil, err
	}
	updated, err := uc.Repo.Update(ctx, plan)
	if err != nil {
		return nil, err
	}
	return toProductionPlanResponse(updated), nil
}
