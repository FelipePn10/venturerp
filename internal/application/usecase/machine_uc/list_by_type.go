package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type ListMachinesByTypeUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *ListMachinesByTypeUseCase) Execute(
	ctx context.Context,
	typeCode int64,
) ([]*response.MachineResponse, error) {
	if !uc.Auth.CanListByType(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListByType(ctx, typeCode)
	if err != nil {
		return nil, err
	}
	return toMachineResponses(list), nil
}
