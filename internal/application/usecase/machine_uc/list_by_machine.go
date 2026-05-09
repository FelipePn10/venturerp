package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type ListItemsByMachineUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *ListItemsByMachineUseCase) Execute(
	ctx context.Context,
	machineCode int64) ([]*entity.ItemMachineTime, error) {
	if !uc.Auth.ListByMachine(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListItemsByMachine(ctx, machineCode)
}
