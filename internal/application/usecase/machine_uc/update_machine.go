package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type UpdateMachineUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *UpdateMachineUseCase) Execute(
	ctx context.Context,
	dto request.UpdateMachineDTO,
) (*response.MachineResponse, error) {
	if !uc.Auth.CanUpdateMachine(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	m := &entity.Machine{
		Code:            dto.Code,
		Name:            dto.Name,
		MachineTypeCode: dto.MachineTypeCode,
		CostCenterCode:  dto.CostCenterCode,
		Capacity:        dto.Capacity,
		CapacityPeriod:  dto.CapacityPeriod,
		CapacityUnit:    dto.CapacityUnit,
		IsActive:        dto.IsActive,
		EfficiencyRate:  dto.EfficiencyRate,
	}

	updated, err := uc.Repo.Update(ctx, m)
	if err != nil {
		return nil, err
	}
	return toMachineResponse(updated), nil
}
