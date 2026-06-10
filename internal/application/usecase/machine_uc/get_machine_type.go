package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type GetMachineTypeUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *GetMachineTypeUseCase) Execute(
	ctx context.Context,
	code int64,
) (*response.MachineTypeResponse, error) {
	if !uc.Auth.CanUpdateMachineType(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	t, err := uc.Repo.GetTypeByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toMachineTypeResponse(t), nil
}
