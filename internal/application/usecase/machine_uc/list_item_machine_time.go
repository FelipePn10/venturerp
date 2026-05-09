package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type ListItemMachineTimesUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *ListItemMachineTimesUseCase) Execute(
	ctx context.Context, itemCode int64,
) ([]*entity.ItemMachineTime, error) {
	if !uc.Auth.CanListItemMachineTimes(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.ListItemMachineTimes(ctx, itemCode)
}
