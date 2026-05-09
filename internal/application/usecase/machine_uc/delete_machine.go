package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type DeleteMachineUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *DeleteMachineUseCase) Execute(ctx context.Context, code int64) error {
	if !uc.Auth.CanDeleteMachine(ctx) {
		return errorsuc.ErrUnauthorized
	}

	return uc.Repo.Delete(ctx, code)
}
