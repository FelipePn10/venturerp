package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type ListMachinesUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *ListMachinesUseCase) Execute(
	ctx context.Context,
) ([]*entity.Machine, error) {
	if !uc.Auth.CanListMachines(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.List(ctx)
}

func (uc *ListMachinesUseCase) GetByCodeMachine(
	ctx context.Context,
	code int64,
) (*entity.MachineType, error) {

	return uc.Repo.GetTypeByCode(ctx, code)
}
