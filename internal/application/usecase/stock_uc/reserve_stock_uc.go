package stock_uc

import (
	"context"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type ReserveStockUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *ReserveStockUseCase) Execute(ctx context.Context, dto request.CreateReservationDTO) (*response.StockReservationResponse, error) {
	if !uc.Auth.CanReserveStock(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	res := &entity.StockReservation{
		ItemCode:          dto.ItemCode,
		Mask:              dto.Mask,
		WarehouseID:       dto.WarehouseID,
		Quantity:          dto.Quantity,
		ReferenceType:     dto.ReferenceType,
		ReferenceCode:     dto.ReferenceCode,
		ReferenceItemCode: dto.ReferenceItemCode,
		Status:            "ACTIVE",
		Notes:             dto.Notes,
		CreatedBy:         userID,
	}

	res.ReservationDate = time.Now()
	if t, msg := datetime.Opcional("data da reserva", dto.ReservationDate); msg != "" {
		return nil, errorsuc.NewValidationError(msg)
	} else if t != nil {
		res.ReservationDate = *t
	}

	if dto.ExpirationDate != nil {
		t, msg := datetime.Opcional("data de validade da reserva", dto.ExpirationDate)
		if msg != "" {
			return nil, errorsuc.NewValidationError(msg)
		}
		res.ExpirationDate = t
	}

	created, err := uc.Repo.CreateReservation(ctx, res)
	if err != nil {
		return nil, err
	}
	return toStockReservationResponse(created), nil
}
