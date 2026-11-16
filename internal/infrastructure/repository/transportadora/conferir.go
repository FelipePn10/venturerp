// Package transportadora confere a transportadora (shipping_carriers) gravada
// em romaneio, carga, pedido de compra e aviso de recebimento.
package transportadora

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type consultaLinha interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Conferir: a FK só garante que o id existe; a transportadora tem de ser da
// empresa do contexto e estar ativa.
func Conferir(ctx context.Context, q consultaLinha, id *int64) error {
	if id == nil {
		return nil
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var ativa bool
	err = q.QueryRow(ctx, `SELECT is_active FROM shipping_carriers WHERE id = $1 AND enterprise_id = $2`, *id, empresa).Scan(&ativa)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewValidationError(fmt.Sprintf("a transportadora %d não existe nesta empresa", *id))
	}
	if err != nil {
		return fmt.Errorf("conferindo a transportadora: %w", err)
	}
	if !ativa {
		return errorsuc.NewValidationError(fmt.Sprintf("a transportadora %d está inativa", *id))
	}
	return nil
}
