package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type DeleteMachineTypeUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *DeleteMachineTypeUseCase) Execute(
	ctx context.Context,
	code int64,
) error {
	if !uc.Auth.CanDeleteMachineType(ctx) {
		return errorsuc.ErrUnauthorized
	}

	return uc.Repo.DeleteType(ctx, code)
}
