package production_order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ModoConferenciaInicio lê o parâmetro de planejamento 52 da empresa (o que
// fazer ao iniciar a OF com pendência). Sem a linha, vale o padrão (AVISAR).
func (r *ProductionOrderRepositoryPGX) ModoConferenciaInicio(ctx context.Context) (string, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return "", err
	}
	var valor string
	err = r.pool.QueryRow(ctx, `SELECT value FROM planning_params WHERE enterprise_id = $1 AND param_number = $2`,
		enterpriseID, entity.ParametroConferenciaInicio).Scan(&valor)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.ConferenciaInicioAvisar, nil
	}
	if err != nil {
		return "", fmt.Errorf("lendo o parâmetro de planejamento %d: %w", entity.ParametroConferenciaInicio, err)
	}
	return entity.ModoConferenciaDoParametro(valor), nil
}

// PendenciasDoInicio junta o que falta para a OF começar: as OFs filhas (pela
// cadeia gravada ao firmar) ainda não concluídas e os materiais a baixar sem
// saldo disponível para ela.
func (r *ProductionOrderRepositoryPGX) PendenciasDoInicio(ctx context.Context, orderID int64) (entity.PendenciasDoInicio, error) {
	var out entity.PendenciasDoInicio
	filhas, err := r.OrdensFilhasPendentes(ctx, orderID)
	if err != nil {
		return out, err
	}
	out.Filhas = filhas
	linhas, err := r.MaterialAvailability(ctx, []int64{orderID})
	if err != nil {
		return out, err
	}
	for _, l := range linhas {
		if l.Falta().IsPositive() {
			out.Faltas = append(out.Faltas, l)
		}
	}
	return out, nil
}

// OrdensFilhasPendentes lista as ordens dos componentes fabricados que
// abastecem a OF e ainda não terminaram: OF aberta/em produção ou ordem
// planejada que ainda não gerou OF.
func (r *ProductionOrderRepositoryPGX) OrdensFilhasPendentes(ctx context.Context, orderID int64) ([]entity.OrdemFilhaPendente, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT filha.code, COALESCE(ofilha.id, 0), COALESCE(ofilha.order_number, 0), filha.item_code,
		       COALESCE(i.business_code::text, ''), COALESCE(i.name, ''),
		       COALESCE(ofilha.status, filha.status::text), COALESCE(ofilha.planned_qty, filha.quantity), COALESCE(ofilha.produced_qty, 0)
		  FROM production_orders pai
		  JOIN production_chain_links l ON l.enterprise_id = pai.enterprise_id
		       AND (l.parent_production_order_id = pai.id
		            OR (l.parent_production_order_id IS NULL AND l.parent_planned_order_id = pai.planned_order_id))
		  JOIN planned_orders filha ON filha.id = l.child_planned_order_id AND filha.enterprise_id = pai.enterprise_id
		       AND filha.is_active AND filha.status::text <> 'CANCELLED'
		  LEFT JOIN LATERAL (SELECT o.id, o.order_number, o.status, o.planned_qty, o.produced_qty
		                       FROM production_orders o
		                      WHERE o.planned_order_id = filha.id AND o.enterprise_id = pai.enterprise_id AND o.is_active
		                      ORDER BY o.id DESC LIMIT 1) ofilha ON TRUE
		  LEFT JOIN items i ON i.code = filha.item_code AND i.enterprise_id = pai.enterprise_id
		 WHERE pai.id = $1 AND pai.enterprise_id = $2
		   AND (ofilha.id IS NULL OR ofilha.status NOT IN ('COMPLETED', 'CLOSED', 'CANCELLED'))
		 ORDER BY 3, 1`, orderID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("lendo as OFs filhas da OF %d: %w", orderID, err)
	}
	defer rows.Close()
	var out []entity.OrdemFilhaPendente
	for rows.Next() {
		var f entity.OrdemFilhaPendente
		if err := rows.Scan(&f.PlannedOrderCode, &f.ProductionOrderID, &f.OrderNumber, &f.ItemCode, &f.BusinessCode, &f.Description,
			&f.Status, &f.Planejada, &f.Produzida); err != nil {
			return nil, fmt.Errorf("lendo a OF filha: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// CadeiaDaOF lê a cadeia de fabricação da OF: os pedidos de venda (da ordem
// planejada dela — inclusive os herdados do pai — e o pedido gravado na OF
// avulsa), as ordens pai que ela abastece e as ordens filhas feitas para ela,
// com o que cada filha deixou reservado para esta OF.
func (r *ProductionOrderRepositoryPGX) CadeiaDaOF(ctx context.Context, orderID int64) (entity.CadeiaDaOF, error) {
	var out entity.CadeiaDaOF
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return out, err
	}
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return out, err
	}
	// O pedido é do comercial: enterprise_code (tenant.Code).
	pedidos, err := r.pool.Query(ctx, `
		SELECT so.code, so.order_number, linha.item_code, linha.sequence,
		       COALESCE(NULLIF(c.trade_name, ''), c.name, '')
		  FROM production_orders o
		  JOIN LATERAL (
		        SELECT soi.sales_order_code, soi.code AS item_code, soi.sequence::int AS sequence
		          FROM sales_order_planned_orders v
		          JOIN sales_order_items soi ON soi.code = v.sales_order_item_code
		         WHERE v.planned_order_id = o.planned_order_id AND v.enterprise_id = o.enterprise_id
		        UNION
		        SELECT o.sales_order_code, NULL::bigint, NULL::int WHERE o.sales_order_code IS NOT NULL
		  ) linha ON TRUE
		  JOIN sales_orders so ON so.code = linha.sales_order_code AND so.enterprise_code = $3
		  LEFT JOIN customers c ON c.code = so.customer_code AND c.enterprise_id = o.enterprise_id
		 WHERE o.id = $1 AND o.enterprise_id = $2
		 ORDER BY so.order_number, linha.sequence NULLS FIRST`, orderID, enterpriseID, enterpriseCode)
	if err != nil {
		return out, fmt.Errorf("lendo os pedidos da OF %d: %w", orderID, err)
	}
	for pedidos.Next() {
		var p entity.PedidoDaCadeia
		if err := pedidos.Scan(&p.SalesOrderCode, &p.SalesOrderNumber, &p.SalesOrderItemCode, &p.Sequence, &p.Customer); err != nil {
			pedidos.Close()
			return out, fmt.Errorf("lendo o pedido da OF: %w", err)
		}
		out.Pedidos = append(out.Pedidos, p)
	}
	pedidos.Close()
	if err := pedidos.Err(); err != nil {
		return out, err
	}

	if out.Pais, err = r.elosDaCadeia(ctx, `
		SELECT DISTINCT pp.code, COALESCE(lof.id, pof.id), COALESCE(lof.order_number, pof.order_number),
		       COALESCE(lof.item_code, pp.item_code), COALESCE(i.business_code::text, ''), COALESCE(i.name, ''),
		       COALESCE(lof.status, pof.status, pp.status::text), COALESCE(lof.planned_qty, pof.planned_qty, pp.quantity),
		       COALESCE(lof.produced_qty, pof.produced_qty, 0),
		       COALESCE((SELECT SUM(rv.quantity) FROM stock_reservations rv
		                  WHERE rv.enterprise_id = o.enterprise_id AND rv.reference_type = 'OF_COMPONENTE'
		                    AND rv.reference_code = COALESCE(lof.id, pof.id) AND rv.item_code = o.item_code
		                    AND rv.status = 'ACTIVE'), 0)
		  FROM production_orders o
		  JOIN production_chain_links l ON l.enterprise_id = o.enterprise_id AND l.child_planned_order_id = o.planned_order_id
		  LEFT JOIN planned_orders pp ON pp.id = l.parent_planned_order_id AND pp.enterprise_id = o.enterprise_id
		  LEFT JOIN production_orders lof ON lof.id = l.parent_production_order_id AND lof.enterprise_id = o.enterprise_id
		  LEFT JOIN LATERAL (SELECT x.id, x.order_number, x.status, x.planned_qty, x.produced_qty FROM production_orders x
		                      WHERE x.planned_order_id = pp.id AND x.enterprise_id = o.enterprise_id AND x.is_active
		                      ORDER BY x.id DESC LIMIT 1) pof ON l.parent_production_order_id IS NULL
		  LEFT JOIN items i ON i.code = COALESCE(lof.item_code, pp.item_code) AND i.enterprise_id = o.enterprise_id
		 WHERE o.id = $1 AND o.enterprise_id = $2
		 ORDER BY 3 NULLS LAST, 1`, orderID, enterpriseID); err != nil {
		return out, fmt.Errorf("lendo as ordens pai da OF %d: %w", orderID, err)
	}
	if out.Filhas, err = r.elosDaCadeia(ctx, `
		SELECT DISTINCT filha.code, ofilha.id, ofilha.order_number, filha.item_code,
		       COALESCE(i.business_code::text, ''), COALESCE(i.name, ''),
		       COALESCE(ofilha.status, filha.status::text), COALESCE(ofilha.planned_qty, filha.quantity), COALESCE(ofilha.produced_qty, 0),
		       COALESCE((SELECT SUM(rv.quantity) FROM stock_reservations rv
		                  WHERE rv.enterprise_id = pai.enterprise_id AND rv.reference_type = 'OF_COMPONENTE'
		                    AND rv.reference_code = pai.id AND rv.item_code = filha.item_code
		                    AND rv.status = 'ACTIVE'), 0)
		  FROM production_orders pai
		  JOIN production_chain_links l ON l.enterprise_id = pai.enterprise_id
		       AND (l.parent_production_order_id = pai.id
		            OR (l.parent_production_order_id IS NULL AND l.parent_planned_order_id = pai.planned_order_id))
		  JOIN planned_orders filha ON filha.id = l.child_planned_order_id AND filha.enterprise_id = pai.enterprise_id
		  LEFT JOIN LATERAL (SELECT x.id, x.order_number, x.status, x.planned_qty, x.produced_qty FROM production_orders x
		                      WHERE x.planned_order_id = filha.id AND x.enterprise_id = pai.enterprise_id AND x.is_active
		                      ORDER BY x.id DESC LIMIT 1) ofilha ON TRUE
		  LEFT JOIN items i ON i.code = filha.item_code AND i.enterprise_id = pai.enterprise_id
		 WHERE pai.id = $1 AND pai.enterprise_id = $2
		 ORDER BY 3 NULLS LAST, 1`, orderID, enterpriseID); err != nil {
		return out, fmt.Errorf("lendo as ordens filhas da OF %d: %w", orderID, err)
	}
	return out, nil
}

func (r *ProductionOrderRepositoryPGX) elosDaCadeia(ctx context.Context, sql string, args ...any) ([]entity.OrdemDaCadeia, error) {
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []entity.OrdemDaCadeia
	for rows.Next() {
		var o entity.OrdemDaCadeia
		if err := rows.Scan(&o.PlannedOrderCode, &o.ProductionOrderID, &o.OrderNumber, &o.ItemCode, &o.BusinessCode, &o.Description,
			&o.Status, &o.Quantity, &o.Produced, &o.Reserved); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
