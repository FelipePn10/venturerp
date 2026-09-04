package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type ListMachineTypesUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *ListMachineTypesUseCase) Execute(
	ctx context.Context,
) ([]*response.MachineTypeResponse, error) {
	if !uc.Auth.CanListTypes(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListTypes(ctx)
	if err != nil {
		return nil, err
	}
	return toMachineTypeResponses(list), nil
}

// GetByCodeType devolve o tipo de máquina — não a máquina.
func (uc *ListMachineTypesUseCase) GetByCodeType(
	ctx context.Context,
	code int64,
) (*response.MachineTypeResponse, error) {

	t, err := uc.Repo.GetTypeByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toMachineTypeResponse(t), nil
}
