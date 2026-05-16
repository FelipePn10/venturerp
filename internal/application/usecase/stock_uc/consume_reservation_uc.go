package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

type ConsumeReservationUseCase struct {
	Repo repository.StockRepository
	Auth ports.AuthService
}

func (uc *ConsumeReservationUseCase) Execute(ctx context.Context, id int64) error {
	if !uc.Auth.CanConsumeReservation(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return uc.Repo.ConsumeReservation(ctx, id)
}
