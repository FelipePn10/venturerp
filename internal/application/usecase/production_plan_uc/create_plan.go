package production_plan_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository"
)

type CreateProductionPlanUseCase struct {
	Repo repository.ProductionPlanRepository
	Auth ports.AuthService
}

func (uc *CreateProductionPlanUseCase) Execute(
	ctx context.Context,
	dto request.CreateProductionPlanDTO,
) (*entity.ProductionPlan, error) {
	if !uc.Auth.CanCreateProductionPlan(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	plan, err := entity.NewProductionPlan(
		dto.Code, dto.Name, dto.IndependentDemands,
		dto.GroupSameDateOrders, dto.PlanningTypes, dto.CreatedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("building production plan: %w", err)
	}
	plan.Classification = dto.Classification
	plan.ClassItemCodes = dto.ClassItemCodes
	plan.OrderItemCode = dto.OrderItemCode
	if dto.Parameters != nil {
		plan.Parameters = dto.Parameters
	}

	return uc.Repo.Create(ctx, plan)
}
