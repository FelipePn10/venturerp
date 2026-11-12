package financial_uc

import (
	"context"
	"fmt"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

// conferirContaBancaria exige que a conta da baixa exista na empresa e esteja
// ativa. Sem isso o pagamento e o recebimento eram gravados em uma conta que
// não existe: o título ficava quitado e o dinheiro não aparecia em saldo
// nenhum.
func conferirContaBancaria(ctx context.Context, repo repository.FinancialRepository, id int64) error {
	if id <= 0 {
		return errorsuc.NewValidationError("informe a conta bancária da baixa")
	}
	conta, err := repo.GetContaBancaria(ctx, id)
	if err != nil {
		if errorsuc.EhNaoEncontrado(err) {
			return errorsuc.NewValidationError(fmt.Sprintf("a conta bancária %d não existe nesta empresa", id))
		}
		return err
	}
	if conta == nil || !conta.IsActive {
		return errorsuc.NewValidationError(fmt.Sprintf("a conta bancária %d está inativa", id))
	}
	return nil
}
