package machine_uc

import (
	"context"
	"errors"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type CreateMachineUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *CreateMachineUseCase) Execute(ctx context.Context, dto request.CreateMachineDTO, userID string) (*entity.Machine, error) {
	if !uc.Auth.CanCreateMachine(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.EfficiencyRate < 0 || dto.EfficiencyRate > 1 {
		return nil, errors.New("efficiency_rate must be between 0.0 and 1.0")
	}

	if dto.Capacity <= 0 {
		return nil, errors.New("capacity must be greater than zero")
	}

	if dto.EfficiencyRate < 0 || dto.EfficiencyRate > 1 {
		return nil, errors.New("efficiency_rate must be between 0.0 and 1.0")
	}

	machineType, err := uc.Repo.GetTypeByCode(
		ctx,
		dto.MachineTypeCode,
	)
	if err != nil {
		return nil, err
	}

	if !machineType.IsActive {
		return nil, errors.New("machine type is inactive")
	}
	m := &entity.Machine{
		Code:            dto.Code,
		Name:            dto.Name,
		MachineTypeCode: dto.MachineTypeCode,
		CostCenterCode:  dto.CostCenterCode,
		Capacity:        dto.Capacity,
		CapacityUnit:    dto.CapacityUnit,
		CapacityPeriod:  dto.CapacityPeriod,
		EfficiencyRate:  dto.EfficiencyRate,
		IsActive:        dto.IsActive,
		CreatedBy:       dto.CreatedBy,
	}
	return uc.Repo.Create(ctx, m)
}
