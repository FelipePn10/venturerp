package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type ListItemMachineTimesUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

func (uc *ListItemMachineTimesUseCase) Execute(
	ctx context.Context, itemCode int64,
) ([]*response.ItemMachineTimeResponse, error) {
	if !uc.Auth.CanListItemMachineTimes(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.ListItemMachineTimes(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toItemMachineTimeResponses(list), nil
}
