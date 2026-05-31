package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type CreateMachineTypeUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *CreateMachineTypeUseCase) Execute(ctx context.Context, dto request.CreateMachineTypeDTO, userID string) (*entity.MachineType, error) {
	if !uc.Auth.CanCreateType(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	mt := &entity.MachineType{
		Code:             dto.Code,
		Name:             dto.Name,
		Description:      dto.Description,
		Type:             dto.Type,
		RequiresOperator: dto.RequiresOperator,
		IsActive:         dto.IsActive,
		CreatedBy:        dto.CreatedBy,
	}
	return uc.Repo.CreateType(ctx, mt)
}
