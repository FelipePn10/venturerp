package overhead_allocation_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/repository"
)

type CreateOverheadAllocationUseCase struct {
	Repo repository.OverheadAllocationRepository
	Auth ports.AuthService
}

func (uc *CreateOverheadAllocationUseCase) Execute(
	ctx context.Context,
	dto request.CreateOverheadAllocationDTO,
) (*response.OverheadAllocationResponse, error) {
	if !uc.Auth.CanCreateOverheadAllocation(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.CostCenterCode == 0 {
		return nil, errorsuc.NewValidationError("cost_center_code is required")
	}
	start, _ := time.Parse("2006-01-02", dto.PeriodStart)
	end, _ := time.Parse("2006-01-02", dto.PeriodEnd)

	oa := &entity.OverheadAllocation{
		CostCenterCode:  dto.CostCenterCode,
		PlanAccountCode: dto.PlanAccountCode,
		AccountCode:     dto.AccountCode,
		PeriodStart:     start,
		PeriodEnd:       end,
		AllocationType:  dto.AllocationType,
		BaseCode:        dto.BaseCode,
		CreatedBy:       dto.CreatedBy,
	}

	result, err := uc.Repo.Create(ctx, oa)
	if err != nil {
		return nil, err
	}

	for _, t := range dto.Targets {
		_, err := uc.Repo.AddTarget(ctx, &entity.AllocationTarget{
			OverheadCode:   result.Code,
			CostCenterCode: t.CostCenterCoed,
			Percentage:     t.Percentage,
			Amount:         t.Amount,
		})
		if err != nil {
			return nil, fmt.Errorf("adding target: %w", err)
		}
	}

	return toOverheadAllocationResponse(result), nil
}
