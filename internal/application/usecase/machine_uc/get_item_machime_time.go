package machine_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/machine/repository"
)

type GetItemMachineTimeUseCase struct {
	Repo repository.MachineRepository
	Auth ports.AuthService
}

//func (uc *GetItemMachineTimeUseCase) Execute(
//	ctx context.Context,
//	code int64) (*entity.ItemMachineTime, error) {
//	if !uc.Auth.CanGetItemMachineTime(ctx) {
//		return nil, errorsuc.ErrUnauthorized
//	}
//
//	return uc.Repo.GetItemMachineTime(ctx, code)
//}
