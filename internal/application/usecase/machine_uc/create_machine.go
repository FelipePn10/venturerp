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
	"github.com/google/uuid"
)

type CreateMachineUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *CreateMachineUseCase) Execute(ctx context.Context, dto request.CreateMachineDTO, userID string) (*response.MachineResponse, error) {
	if !uc.Auth.CanCreateMachine(ctx) {
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

	// O autor é sempre o usuário autenticado, nunca o corpo da requisição.
	authenticatedUserID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if authenticatedUserID == uuid.Nil {
		return nil, errorsuc.NewValidationError("não foi possível identificar o usuário da sessão")
	}

	machineType, err := uc.Repo.GetTypeByCode(ctx, dto.MachineTypeCode)
	if err != nil || machineType == nil {
		return nil, errorsuc.NewNotFoundError(
			fmt.Sprintf("tipo de máquina %d não encontrado nesta empresa", dto.MachineTypeCode))
	}
	if !machineType.IsActive {
		return nil, errorsuc.NewValidationError(
			fmt.Sprintf("o tipo de máquina %s está inativo", machineType.Name))
	}
	if existing, getErr := uc.Repo.GetByCode(ctx, dto.Code); getErr == nil && existing != nil {
		return nil, errorsuc.NewConflictError(fmt.Sprintf("já existe uma máquina com o código %d", dto.Code))
	}

	m := &entity.Machine{
		Code:            dto.Code,
		Name:            strings.TrimSpace(dto.Name),
		MachineTypeCode: dto.MachineTypeCode,
		CostCenterCode:  dto.CostCenterCode,
		Capacity:        dto.Capacity,
		CapacityUnit:    capacityUnit,
		CapacityPeriod:  capacityPeriod,
		EfficiencyRate:  efficiency,
		IsActive:        dto.IsActive,
		CreatedBy:       authenticatedUserID,
	}
	created, err := uc.Repo.Create(ctx, m)
	if err != nil {
		return nil, err
	}
	return toMachineResponse(created), nil
}
