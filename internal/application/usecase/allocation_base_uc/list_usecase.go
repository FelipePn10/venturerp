package allocation_base_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/allocation_base/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/allocation_base/repository"
)

type ListAllocationBasesUseCase struct {
	Repo repository.AllocationBaseRepository
	Auth ports.AuthService
}

func (uc *ListAllocationBasesUseCase) Execute(
	ctx context.Context,
) ([]*entity.AllocationBase, error) {
	if !uc.Auth.ListAllocation(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	return uc.Repo.List(ctx)
}
