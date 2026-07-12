package production_plan_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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
) (*response.ProductionPlanResponse, error) {
	if !uc.Auth.CanCreateProductionPlan(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	createdBy, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	plan, err := entity.NewProductionPlan(
		dto.Code, dto.Name, dto.IndependentDemands,
		dto.GroupSameDateOrders, dto.PlanningTypes, createdBy,
	)
	if err != nil {
		return nil, fmt.Errorf("building production plan: %w", err)
	}
	if err := plan.Configure(dto.Classification, dto.ClassItemCodes, dto.OrderItemCode, dto.Parameters); err != nil {
		return nil, err
	}

	created, err := uc.Repo.Create(ctx, plan)
	if err != nil {
		return nil, err
	}
	return toProductionPlanResponse(created), nil
}
