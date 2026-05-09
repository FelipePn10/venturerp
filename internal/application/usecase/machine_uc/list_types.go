package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type ListMachineTypesUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *ListMachineTypesUseCase) Execute(
	ctx context.Context,
) ([]*entity.MachineType, error) {
	if !uc.Auth.CanListTypes(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListTypes(ctx)
}

func (uc *ListMachineTypesUseCase) GetByCodeType(
	ctx context.Context,
	code int64,
) (*entity.Machine, error) {

	return uc.Repo.GetByCode(ctx, code)
}
