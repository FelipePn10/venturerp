package mrp_calculation

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/shopspring/decimal"
)

// ProducaoAguardandoRetorno é, por item com componente fornecido pelo cliente, o
// que as OFs entregaram menos o que os pedidos já atenderam: a peça pronta que
// ainda vai sair pela nota de retorno. Pedido é do comercial e filtra por
// enterprise_code; OF filtra por enterprise_id.
func (r *MRPCalculationRepositorySQLC) ProducaoAguardandoRetorno(ctx context.Context) (map[int64]decimal.Decimal, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		WITH entregue AS (
			SELECT o.item_code, SUM(o.produced_qty) AS qtd
			  FROM production_orders o
			 WHERE o.enterprise_id = $1 AND o.is_active AND o.status <> 'CANCELLED'
			   AND EXISTS (SELECT 1 FROM item_structures s
			                WHERE s.parent_code = o.item_code AND s.is_active AND s.customer_supplied)
			 GROUP BY o.item_code
		), atendido AS (
			SELECT soi.item_code, SUM(soi.attended_qty) AS qtd
			  FROM sales_order_items soi
			  JOIN sales_orders so ON so.code = soi.sales_order_code
			 WHERE so.enterprise_code = $2 AND soi.item_code IN (SELECT item_code FROM entregue)
			 GROUP BY soi.item_code
		)
		SELECT e.item_code, e.qtd - COALESCE(a.qtd, 0)
		  FROM entregue e
		  LEFT JOIN atendido a ON a.item_code = e.item_code
		 WHERE e.qtd > COALESCE(a.qtd, 0)`, enterpriseID, enterpriseCode)
	if err != nil {
		return nil, fmt.Errorf("lendo a produção de beneficiamento a devolver: %w", err)
	}
	defer rows.Close()
	out := make(map[int64]decimal.Decimal)
	for rows.Next() {
		var item int64
		var quantidade decimal.Decimal
		if err := rows.Scan(&item, &quantidade); err != nil {
			return nil, err
		}
		out[item] = quantidade
	}
	return out, rows.Err()
}
