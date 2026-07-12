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

type ManageProductionPlanInterFactoriesUseCase struct {
	Repo repository.ProductionPlanRepository
	Auth ports.AuthService
}

func (uc *ManageProductionPlanInterFactoriesUseCase) Replace(ctx context.Context, planCode int64, dto request.ReplaceProductionPlanInterFactoriesDTO) ([]*response.ProductionPlanInterFactoryResponse, error) {
	if !uc.Auth.CanUpdateProductionPlan(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if planCode <= 0 {
		return nil, fmt.Errorf("%w: plan code must be positive", entity.ErrInvalidPlan)
	}
	if _, err := uc.Repo.GetByCode(ctx, planCode); err != nil {
		return nil, err
	}
	seen := make(map[int64]struct{}, len(dto.Enterprises))
	entries := make([]*entity.InterFactoryEnterprise, 0, len(dto.Enterprises))
	for _, item := range dto.Enterprises {
		if item.EnterpriseCode <= 0 {
			return nil, fmt.Errorf("%w: inter-factory enterprise code must be positive", entity.ErrInvalidPlan)
		}
		if _, ok := seen[item.EnterpriseCode]; ok {
			return nil, fmt.Errorf("%w: duplicated inter-factory enterprise %d", entity.ErrInvalidPlan, item.EnterpriseCode)
		}
		seen[item.EnterpriseCode] = struct{}{}
		entries = append(entries, &entity.InterFactoryEnterprise{EnterpriseCode: item.EnterpriseCode, AutoRelease: item.AutoRelease})
	}
	result, err := uc.Repo.ReplaceInterFactories(ctx, planCode, entries)
	if err != nil {
		return nil, err
	}
	return interFactoryResponses(result), nil
}

func (uc *ManageProductionPlanInterFactoriesUseCase) List(ctx context.Context, planCode int64) ([]*response.ProductionPlanInterFactoryResponse, error) {
	if !uc.Auth.CanListProductionPlans(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if _, err := uc.Repo.GetByCode(ctx, planCode); err != nil {
		return nil, err
	}
	items, err := uc.Repo.ListInterFactories(ctx, planCode)
	if err != nil {
		return nil, err
	}
	return interFactoryResponses(items), nil
}

func interFactoryResponses(items []*entity.InterFactoryEnterprise) []*response.ProductionPlanInterFactoryResponse {
	out := make([]*response.ProductionPlanInterFactoryResponse, 0, len(items))
	for _, item := range items {
		out = append(out, &response.ProductionPlanInterFactoryResponse{EnterpriseCode: item.EnterpriseCode, EnterpriseName: item.EnterpriseName, AutoRelease: item.AutoRelease})
	}
	return out
}
