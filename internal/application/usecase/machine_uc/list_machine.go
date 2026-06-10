package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type ListMachinesUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *ListMachinesUseCase) Execute(
	ctx context.Context,
) ([]*response.MachineResponse, error) {
	if !uc.Auth.CanListMachines(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return toMachineResponses(list), nil
}

func (uc *ListMachinesUseCase) GetByCodeMachine(
	ctx context.Context,
	code int64,
) (*response.MachineTypeResponse, error) {

	t, err := uc.Repo.GetTypeByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toMachineTypeResponse(t), nil
}
