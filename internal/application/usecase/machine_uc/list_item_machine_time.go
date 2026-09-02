package machine_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	itemrepo "github.com/FelipePn10/panossoerp/internal/domain/items/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type ListItemMachineTimesUseCase struct {
	Repo     repository.MachineRepository
	ItemRepo itemrepo.ItemRepository
	Auth     ports.AuthService
}

func (uc *ListItemMachineTimesUseCase) Execute(
	ctx context.Context, publicItemCode request.TextCode,
) ([]*response.ItemMachineTimeResponse, error) {
	if !uc.Auth.CanListItemMachineTimes(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	itemCode := int64(0)
	if publicItemCode.String() != "" {
		item, err := itemresolution.Resolve(ctx, uc.ItemRepo, publicItemCode)
		if err != nil {
			return nil, err
		}
		itemCode = int64(item.Code)
	}
	list, err := uc.Repo.ListItemMachineTimes(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toItemMachineTimeResponses(list), nil
}
