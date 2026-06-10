package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type GetMachineUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *GetMachineUseCase) Execute(
	ctx context.Context,
	code int64,
) (*response.MachineResponse, error) {
	if !uc.Auth.CanGetMachine(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	m, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toMachineResponse(m), nil
}
