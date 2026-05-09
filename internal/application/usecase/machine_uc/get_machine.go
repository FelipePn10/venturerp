package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type GetMachineUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *GetMachineUseCase) Execute(
	ctx context.Context,
	code int64,
) (*entity.Machine, error) {
	if !uc.Auth.CanGetMachine(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.GetByCode(ctx, code)
}
