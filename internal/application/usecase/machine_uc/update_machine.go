package machine_uc

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
	"github.com/FelipePn10/panossoerp/internal/shared/ptrutil"
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
	if err := validateMachineFields(dto.Code, dto.Name, dto.MachineTypeCode, dto.Capacity); err != nil {
		return nil, err
	}
	capacityUnit, err := normalizeCapacityUnit(dto.CapacityUnit)
	if err != nil {
		return nil, err
	}
	capacityPeriod, err := normalizeCapacityPeriod(dto.CapacityPeriod)
	if err != nil {
		return nil, err
	}
	efficiency, err := normalizeEfficiency(dto.EfficiencyRate)
	if err != nil {
		return nil, err
	}
	machineType, err := uc.Repo.GetTypeByCode(ctx, dto.MachineTypeCode)
	if err != nil || machineType == nil {
		return nil, errorsuc.NewNotFoundError(
			fmt.Sprintf("tipo de máquina %d não encontrado nesta empresa", dto.MachineTypeCode))
	}

	m := &entity.Machine{
		Code:            dto.Code,
		Name:            strings.TrimSpace(dto.Name),
		MachineTypeCode: dto.MachineTypeCode,
		CostCenterCode:  dto.CostCenterCode,
		Capacity:        dto.Capacity,
		CapacityPeriod:  capacityPeriod,
		CapacityUnit:    capacityUnit,
		IsActive:        ptrutil.BoolOrTrue(dto.IsActive),
		EfficiencyRate:  efficiency,
	}

	if err := camposDeCadastro(m, camposOpcionais{
		ResourceGroupID: dto.ResourceGroupID, CalendarID: dto.CalendarID, Location: dto.Location,
		IsCritical: dto.IsCritical, UsageDescription: dto.UsageDescription, AcquiredOn: dto.AcquiredOn,
		PreparationTime: dto.PreparationTime, PreparationTimeUnit: dto.PreparationTimeUnit,
		SupplierCode: dto.SupplierCode, Brand: dto.Brand, IsPreferred: dto.IsPreferred,
		MaintenanceResponsibleEmployeeID: dto.MaintenanceResponsibleEmployeeID,
	}); err != nil {
		return nil, err
	}

	updated, err := uc.Repo.Update(ctx, m)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("máquina %d não encontrada nesta empresa", dto.Code))
	}
	return toMachineResponse(updated), nil
}
