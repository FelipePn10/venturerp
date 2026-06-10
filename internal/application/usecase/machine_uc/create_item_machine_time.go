package machine_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
	machinesvc "github.com/FelipePn10/panossoerp/internal/domain/machine/service"
)

type CreateItemMachineTimeUseCase struct {
	Repo     repository.MachineRepository
	ItemRepo itemrepo.ItemRepository
	Auth     ports.AuthService
}

func (uc *CreateItemMachineTimeUseCase) Execute(
	ctx context.Context, dto request.CreateItemMachineTimeDTO,
) (*response.ItemMachineTimeResponse, error) {
	if !uc.Auth.CanCreateItemTimeMachine(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	if err := uc.validateUnitCompatibility(ctx, dto.ItemCode, dto.MachineCode); err != nil {
		return nil, err
	}

	imt := &entity.ItemMachineTime{
		ItemCode:           dto.ItemCode,
		Mask:               dto.Mask,
		MachineCode:        dto.MachineCode,
		ProductionTime:     dto.ProductionTime,
		ProductionTimeUnit: dto.ProductionTimeUnit,
		ProductionBaseQty:  dto.ProductionBaseQty,
		SetupTime:          dto.SetupTime,
		Priority:           dto.Priority,
	}
	created, err := uc.Repo.CreateItemMachineTime(ctx, imt)
	if err != nil {
		return nil, err
	}
	return toItemMachineTimeResponse(created), nil
}

func (uc *CreateItemMachineTimeUseCase) GetByCodeTime(
	ctx context.Context,
	code int64,
) (*response.MachineResponse, error) {
	m, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toMachineResponse(m), nil
}

func (uc *CreateItemMachineTimeUseCase) validateUnitCompatibility(
	ctx context.Context,
	itemCode int64,
	machineCode int64,
) error {
	itemCodeVO, err := valueobject.NewItemCode(itemCode)
	if err != nil {
		return fmt.Errorf("item code is invalid: %w", err)
	}

	item, err := uc.ItemRepo.FindItemByCode(ctx, itemCodeVO)
	if err != nil {
		return fmt.Errorf("item %d not found: %w", itemCode, err)
	}

	machine, err := uc.Repo.GetByCode(ctx, machineCode)
	if err != nil {
		return fmt.Errorf("machine %d not found: %w", machineCode, err)
	}

	_, err = machinesvc.CheckUnitCompatibility(
		item.Warehouse.UnitOfMeasurement,
		machine.CapacityUnit,
	)
	if err != nil {
		return fmt.Errorf(
			"invalid configuration — item '%d' uses unit '%s' but machine '%d' operates on '%s': %w",
			itemCode, item.Warehouse.UnitOfMeasurement,
			machineCode, machine.CapacityUnit,
			err,
		)
	}

	return nil
}
