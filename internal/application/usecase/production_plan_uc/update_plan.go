package production_plan_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
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

	plan := &entity.ProductionPlan{
		Code:                dto.Code,
		Name:                dto.Name,
		IndependentDemands:  dto.IndependentDemands,
		GroupSameDateOrders: dto.GroupSameDateOrders,
		PlanningTypes:       dto.PlanningTypes,
		Classification:      dto.Classification,
		ClassItemCodes:      dto.ClassItemCodes,
		OrderItemCode:       dto.OrderItemCode,
		Parameters:          dto.Parameters,
	}
	updated, err := uc.Repo.Update(ctx, plan)
	if err != nil {
		return nil, err
	}
	return toProductionPlanResponse(updated), nil
}
