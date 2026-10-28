// Package parametroscomerciais: parâmetros comerciais da empresa (Fase 5.3).
package parametroscomerciais

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Parametros struct {
	// DescontoMaxPct: desconto máximo da linha do pedido sem aprovação (nil = sem limite).
	DescontoMaxPct *decimal.Decimal `json:"desconto_max_pct"`
}

type Repository interface {
	Obter(ctx context.Context) (Parametros, error)
	Salvar(ctx context.Context, p Parametros, por uuid.UUID) error
	DescontoMaximo(ctx context.Context) (*decimal.Decimal, error)
}
