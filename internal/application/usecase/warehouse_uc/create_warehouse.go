package warehouse_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/warehouse/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/warehouse/repository"
)

type CreateWarehouseUseCase struct {
	Repo repository.WarehouseRepository
	Auth ports.AuthService
}

func NewCreateWarehouseUseCase(
	repo repository.WarehouseRepository,
	auth ports.AuthService,
) *CreateWarehouseUseCase {
	return &CreateWarehouseUseCase{
		Repo: repo,
		Auth: auth,
	}
}

func (uc *CreateWarehouseUseCase) Execute(
	ctx context.Context,
	dto request.CreateWarehouseRequestDTO,
) (*entity.Warehouse, error) {
	if !uc.Auth.CanCreateWarehouse(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	warehouse, err := entity.NewWarehouse(
		dto.Code,
		dto.Description,
		dto.Location,
		dto.Type,
		dto.Disposition,
		dto.ReservationsAllowed,
		dto.CreatedBy,
	)
	if err != nil {
		return nil, err
	}

	create, err := uc.Repo.Create(ctx, warehouse)
	if err != nil {
		return nil, err
	}
	return create, nil
}
