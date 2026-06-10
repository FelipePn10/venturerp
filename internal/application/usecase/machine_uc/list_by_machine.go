package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type ListItemsByMachineUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *ListItemsByMachineUseCase) Execute(
	ctx context.Context,
	machineCode int64) ([]*response.ItemMachineTimeResponse, error) {
	if !uc.Auth.ListByMachine(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListItemsByMachine(ctx, machineCode)
	if err != nil {
		return nil, err
	}
	return toItemMachineTimeResponses(list), nil
}
