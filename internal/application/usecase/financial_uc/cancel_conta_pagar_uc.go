package financial_uc

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type CancelContaPagarUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CancelContaPagarUseCase) Execute(ctx context.Context, id int64) error {
	if !uc.Auth.CanCancelContaPagar(ctx) {
		return errorsuc.ErrUnauthorized
	}
	// Só se cancela o que ainda não foi pago. Cancelar título quitado tirava a
	// dívida do sistema com o dinheiro já fora do banco: o pagamento ficava no
	// fluxo de caixa sem título que o explicasse.
	cp, err := uc.Repo.GetContaPagar(ctx, id)
	if err != nil {
		return err
	}
	if msg := impedeCancelarTitulo(string(cp.Status), cp.ValorPago.Add(cp.ValorAdiantamentoAbatido), "pago", "pagamento"); msg != "" {
		return errorsuc.NewValidationError(msg)
	}
	return uc.Repo.CancelContaPagar(ctx, id)
}
