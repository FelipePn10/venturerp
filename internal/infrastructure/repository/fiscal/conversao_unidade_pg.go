package fiscal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// FatorDeConversao resolve quanto 1 unidade `de` vale na unidade `para`.
//
// A ordem é a do cadastro mais específico para o mais geral: conversão do item
// na direção pedida, a inversa dela, a conversão global da empresa e a inversa.
// A inversa é calculada no banco, em NUMERIC: 1/6000 tem de sair com as dez
// casas do fator, não com o arredondamento de um float.
func (r *FiscalRepositoryPG) FatorDeConversao(ctx context.Context, itemCode int64, de, para string) (*decimal.Decimal, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	de, para = strings.ToUpper(strings.TrimSpace(de)), strings.ToUpper(strings.TrimSpace(para))
	if de == "" || para == "" {
		return nil, nil
	}
	if de == para {
		um := decimal.NewFromInt(1)
		return &um, nil
	}
	var f decimal.Decimal
	err = r.db.QueryRow(ctx, `
		SELECT fator FROM (
		    SELECT c.factor::numeric AS fator, 1 AS ordem
		      FROM item_unit_conversions c JOIN items i ON i.code = c.item_code AND i.enterprise_id = $1
		     WHERE c.item_code = $2 AND c.is_active AND c.mask = '' AND upper(c.from_uom) = $3 AND upper(c.to_uom) = $4
		    UNION ALL
		    SELECT round(1 / c.factor::numeric, 10), 2
		      FROM item_unit_conversions c JOIN items i ON i.code = c.item_code AND i.enterprise_id = $1
		     WHERE c.item_code = $2 AND c.is_active AND c.mask = '' AND upper(c.from_uom) = $4 AND upper(c.to_uom) = $3
		    UNION ALL
		    SELECT g.factor::numeric, 3 FROM global_unit_conversions g
		     WHERE g.enterprise_id = $1 AND g.is_active AND upper(g.from_uom) = $3 AND upper(g.to_uom) = $4
		    UNION ALL
		    SELECT round(1 / g.factor::numeric, 10), 4 FROM global_unit_conversions g
		     WHERE g.enterprise_id = $1 AND g.is_active AND upper(g.from_uom) = $4 AND upper(g.to_uom) = $3
		) x ORDER BY ordem LIMIT 1`, empresa, itemCode, de, para).Scan(&f)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lendo a conversão %s→%s do item %d: %w", de, para, itemCode, err)
	}
	return &f, nil
}
