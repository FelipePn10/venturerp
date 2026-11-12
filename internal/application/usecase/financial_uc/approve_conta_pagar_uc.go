package financial_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type ApproveContaPagarUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *ApproveContaPagarUseCase) Execute(ctx context.Context, id int64) error {
	if !uc.Auth.CanApproveContaPagar(ctx) {
		return errorsuc.ErrUnauthorized
	}

	userID, err := uc.Auth.UserID(ctx)
	if err != nil {
		return err
	}

	// Só o título que ainda aguarda aprovação é aprovado. Aprovar o pago ou o
	// cancelado o devolvia para "aprovado": a dívida quitada voltava a constar
	// em aberto e podia ser paga de novo.
	cp, err := uc.Repo.GetContaPagar(ctx, id)
	if err != nil {
		return err
	}
	if cp.Status != entity.ContaPagarStatusPendente && cp.Status != entity.ContaPagarStatusVencido {
		return errorsuc.NewValidationError(fmt.Sprintf("só título pendente ou vencido pode ser aprovado; este está %s", cp.Status))
	}
	return uc.Repo.ApproveContaPagar(ctx, id, userID)
}
