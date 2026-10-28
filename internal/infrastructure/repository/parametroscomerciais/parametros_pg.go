// Package parametroscomerciais grava os parâmetros comerciais da empresa da
// sessão (alçada de desconto da linha do pedido).
package parametroscomerciais

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	dom "github.com/FelipePn10/panossoerp/internal/domain/parametroscomerciais"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type Parametros = dom.Parametros

type PG struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *PG { return &PG{pool: pool} }

func (r *PG) Obter(ctx context.Context) (Parametros, error) {
	var p Parametros
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return p, err
	}
	err = r.pool.QueryRow(ctx, `SELECT desconto_max_pct FROM parametros_comerciais WHERE enterprise_id=$1`, empresa).Scan(&p.DescontoMaxPct)
	if errors.Is(err, pgx.ErrNoRows) {
		return Parametros{}, nil
	}
	return p, err
}

// DescontoMaximo: nil = sem limite.
func (r *PG) DescontoMaximo(ctx context.Context) (*decimal.Decimal, error) {
	p, err := r.Obter(ctx)
	return p.DescontoMaxPct, err
}

func (r *PG) Salvar(ctx context.Context, p Parametros, por uuid.UUID) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO parametros_comerciais (enterprise_id, desconto_max_pct, updated_by) VALUES ($1,$2,$3)
		ON CONFLICT (enterprise_id) DO UPDATE SET desconto_max_pct=EXCLUDED.desconto_max_pct, updated_by=EXCLUDED.updated_by, updated_at=NOW()`,
		empresa, p.DescontoMaxPct, por)
	return err
}
