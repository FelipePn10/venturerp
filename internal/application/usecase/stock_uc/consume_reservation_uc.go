package stock_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
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
	if err := conferirReservaAvulsa(ctx, uc.Repo, id); err != nil {
		return err
	}
	return uc.Repo.ConsumeReservation(ctx, id)
}

// conferirReservaAvulsa: a reserva da onda, da demanda e da requisição são do
// documento. Encerrada à parte, o saldo do lote e a linha da onda ficam presos.
func conferirReservaAvulsa(ctx context.Context, repo repository.StockRepository, id int64) error {
	r, err := repo.GetReservation(ctx, id)
	if err != nil {
		return err
	}
	switch r.ReferenceType {
	case "ONDA_SEPARACAO":
		return errorsuc.NewValidationError("esta reserva é de uma onda de separação: confirme ou cancele a onda")
	case entity.ReferenceTypeIndependentDemand:
		return errorsuc.NewValidationError("esta reserva é de uma demanda independente: atenda ou encerre a demanda")
	case entity.ReferenceTypeRequisition:
		return errorsuc.NewValidationError("esta reserva é de uma requisição: atenda ou cancele a requisição")
	}
	return nil
}
