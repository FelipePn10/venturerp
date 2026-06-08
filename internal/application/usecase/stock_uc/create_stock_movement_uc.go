package stock_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type CreateStockMovementUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *CreateStockMovementUseCase) Execute(ctx context.Context, dto request.CreateStockMovementDTO) (*entity.StockMovement, error) {
	if !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	m := &entity.StockMovement{
		ItemCode:      dto.ItemCode,
		Mask:          dto.Mask,
		WarehouseID:   dto.WarehouseID,
		MovementType:  dto.MovementType,
		Quantity:      dto.Quantity,
		UnitPrice:     dto.UnitPrice,
		TotalPrice:    dto.TotalPrice,
		ReferenceType: dto.ReferenceType,
		ReferenceCode: dto.ReferenceCode,
		Lot:           dto.Lot,
		SerialNumber:  dto.SerialNumber,
		Batch:         dto.Batch,
		Notes:         dto.Notes,
		CreatedBy:     userID,
	}

	if dto.ExpirationDate != nil {
		t, _ := time.Parse("2006-01-02", *dto.ExpirationDate)
		m.ExpirationDate = &t
	}

	return uc.Repo.CreateMovement(ctx, m)
}
