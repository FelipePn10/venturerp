package allocation_base_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/allocation_base/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/allocation_base/repository"
)

type CreateAllocationBaseUseCase struct {
	Repo repository.AllocationBaseRepository
	Auth ports.AuthService
}

func (uc *CreateAllocationBaseUseCase) Execute(
	ctx context.Context,
	dto request.CreateAllocationBaseDTO,
) (*entity.AllocationBase, error) {
	if !uc.Auth.CreateAllocation(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	ab := &entity.AllocationBase{
		Code:        dto.Code,
		Description: dto.Description,
		Period:      dto.Period,
		Observation: dto.Observation,
		CreatedBy:   dto.CreatedBy,
	}

	result, err := uc.Repo.Create(ctx, ab)
	if err != nil {
		return nil, err
	}

	for _, item := range dto.Items {
		_, err := uc.Repo.AddItem(ctx, &entity.AllocationBaseItem{
			AllocationBaseCode: result.Code,
			CostCenterCode:     item.CostCenterCode,
			Amount:             item.Amount,
			Percentage:         item.Percentage,
		})
		if err != nil {
			return nil, fmt.Errorf("adding base item: %w", err)
		}
	}

	return result, nil
}
