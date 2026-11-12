package financial_uc

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/financial/repository"
)

type CancelContaReceberUseCase struct {
	Repo repository.FinancialRepository
	Auth ports.AuthService
}

func (uc *CancelContaReceberUseCase) Execute(ctx context.Context, id int64) error {
	if !uc.Auth.CanCancelContaReceber(ctx) {
		return errorsuc.ErrUnauthorized
	}
	// Só se cancela o que ainda não foi recebido (ver CancelContaPagar).
	cr, err := uc.Repo.GetContaReceber(ctx, id)
	if err != nil {
		return err
	}
	if msg := impedeCancelarTitulo(string(cr.Status), cr.ValorRecebido, "recebido", "recebimento"); msg != "" {
		return errorsuc.NewValidationError(msg)
	}
	return uc.Repo.CancelContaReceber(ctx, id)
}

// impedeCancelarTitulo devolve por que o título não pode ser cancelado (vazio
// quando pode): tem de estar em aberto e sem nenhuma baixa.
func impedeCancelarTitulo(status string, baixado decimal.Decimal, feito, baixa string) string {
	switch status {
	case "PENDENTE", "APROVADO", "VENCIDO":
	default:
		return fmt.Sprintf("só título em aberto pode ser cancelado; este está %s", status)
	}
	if baixado.IsPositive() {
		return fmt.Sprintf("o título já tem %s de %s: estorne o %s antes de cancelar", feito, baixado.StringFixed(2), baixa)
	}
	return ""
}
