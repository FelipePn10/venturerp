package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type ListMachinesByTypeUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *ListMachinesByTypeUseCase) Execute(
	ctx context.Context,
	typeCode int64,
) ([]*entity.Machine, error) {
	if !uc.Auth.CanListByType(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListByType(ctx, typeCode)
}
