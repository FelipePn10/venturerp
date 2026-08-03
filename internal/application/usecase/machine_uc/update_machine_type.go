package machine_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type UpdateMachineTypeUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *UpdateMachineTypeUseCase) Execute(
	ctx context.Context,
	dto request.UpdateMachineTypeDTO,
) (*response.MachineTypeResponse, error) {
	if !uc.Auth.CanUpdateMachineType(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if !dto.Type.IsValid() {
		return nil, fmt.Errorf("invalid machine type: %s", dto.Type)
	}

	mt := &entity.MachineType{
		Code:             dto.Code,
		Name:             dto.Name,
		Description:      dto.Description,
		Type:             dto.Type,
		RequiresOperator: dto.RequiresOperator,
		IsActive:         dto.IsActive,
	}

	updated, err := uc.Repo.UpdateType(ctx, mt)
	if err != nil {
		return nil, err
	}
	return toMachineTypeResponse(updated), nil
}
