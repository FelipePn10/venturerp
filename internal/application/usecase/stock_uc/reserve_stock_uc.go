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

// Listar devolve as reservas da empresa: do item, quando informado, e só as
// ativas, quando pedido. Sem isto a tela só agia sobre um número digitado.
func (uc *ReserveStockUseCase) Listar(ctx context.Context, itemCode int64, soAtivas bool) ([]*response.StockReservationResponse, error) {
	if !uc.Auth.CanGetStockBalance(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	var lista []*entity.StockReservation
	var err error
	switch {
	case itemCode > 0:
		lista, err = uc.Repo.ListReservationsByItem(ctx, itemCode)
	case soAtivas:
		lista, err = uc.Repo.ListActiveReservations(ctx)
	default:
		lista, err = uc.Repo.ListReservations(ctx)
	}
	if err != nil {
		return nil, err
	}
	out := make([]*response.StockReservationResponse, 0, len(lista))
	for _, r := range lista {
		if soAtivas && r.Status != "ACTIVE" {
			continue
		}
		out = append(out, toStockReservationResponse(r))
	}
	return out, nil
}

func (uc *ReserveStockUseCase) Execute(ctx context.Context, dto request.CreateReservationDTO) (*response.StockReservationResponse, error) {
	if !uc.Auth.CanReserveStock(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if !dto.Quantity.IsPositive() {
		return nil, errorsuc.NewValidationError("a quantidade da reserva precisa ser maior que zero")
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
