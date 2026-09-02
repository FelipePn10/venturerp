package machine_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
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

	item, err := itemresolution.Resolve(ctx, uc.ItemRepo, dto.ItemCode)
	if err != nil {
		return nil, err
	}
	itemCode := int64(item.Code)

	if err := uc.validateUnitCompatibility(ctx, item, dto.MachineCode); err != nil {
		return nil, err
	}

	imt := &entity.ItemMachineTime{
		ItemCode:           itemCode,
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
	item *itementity.Item,
	machineCode int64,
) error {
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
			"invalid configuration — item '%s' uses unit '%s' but machine '%d' operates on '%s': %w",
			item.BusinessCode, item.Warehouse.UnitOfMeasurement,
			machineCode, machine.CapacityUnit,
			err,
		)
	}

	return nil
}
