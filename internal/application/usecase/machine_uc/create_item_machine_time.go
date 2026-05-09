package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type CreateItemMachineTimeUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *CreateItemMachineTimeUseCase) Execute(
	ctx context.Context, dto request.CreateItemMachineTimeDTO,
) (*entity.ItemMachineTime, error) {
	if !uc.Auth.CanCreateItemTimeMachine(ctx) {
		return nil, errorsuc.ErrUnauthorized
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
	return uc.Repo.CreateItemMachineTime(ctx, imt)
}

func (uc *CreateItemMachineTimeUseCase) GetByCodeTime(
	ctx context.Context,
	code int64,
) (*entity.Machine, error) {

	return uc.Repo.GetByCode(ctx, code)
}
