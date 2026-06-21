package warehouse_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/warehouse/repository"
)

type GetWarehouseUseCase struct {
	Repo repository.WarehouseRepository
	Auth ports.AuthService
}

func NewGetWarehouseUseCase(
	repo repository.WarehouseRepository,
	auth ports.AuthService,
) *GetWarehouseUseCase {
	return &GetWarehouseUseCase{Repo: repo, Auth: auth}
}

func (uc *GetWarehouseUseCase) Execute(ctx context.Context, code string) (*response.WarehouseResponse, error) {
	if !uc.Auth.CanCreateWarehouse(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	w, err := uc.Repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	return toWarehouseResponse(w), nil
}
