package warehouse_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/warehouse/repository"
)

type ListWarehousesUseCase struct {
	Repo repository.WarehouseRepository
	Auth ports.AuthService
}

func NewListWarehousesUseCase(
	repo repository.WarehouseRepository,
	auth ports.AuthService,
) *ListWarehousesUseCase {
	return &ListWarehousesUseCase{Repo: repo, Auth: auth}
}

func (uc *ListWarehousesUseCase) Execute(ctx context.Context) ([]*response.WarehouseResponse, error) {
	if !uc.Auth.CanCreateWarehouse(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	list, err := uc.Repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*response.WarehouseResponse, 0, len(list))
	for _, w := range list {
		out = append(out, toWarehouseResponse(w))
	}
	return out, nil
}
