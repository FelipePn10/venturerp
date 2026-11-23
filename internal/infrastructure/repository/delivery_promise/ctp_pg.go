package delivery_promise

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_promise/service"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// SituacaoParaPromessa lê estoque livre, entradas e compromissos abertos,
// prazos e a fila de capacidade do item da empresa da sessão.
func (r *Repository) SituacaoParaPromessa(ctx context.Context, item int64) (service.SituacaoDoItem, error) {
	s := service.SituacaoDoItem{ItemCode: item}
	id, err := tenant.ID(ctx)
	if err != nil {
		return s, err
	}
	code, err := tenant.Code(ctx)
	if err != nil {
		return s, err
	}
	var tipo *int16
	var fab, compra, fornecedor *int
	err = r.pool.QueryRow(ctx, `
		SELECT i.engineering_type, i.planning_manufacturing_lead_time_days, i.planning_purchase_lead_time_days,
		       (SELECT MIN(p.lead_time_days) FROM item_preferred_suppliers p WHERE p.enterprise_id = $1 AND p.item_code = i.code AND p.is_active AND NOT p.blocked),
		       COALESCE((SELECT SUM(GREATEST(b.quantity - b.reserved_qty, 0)) FROM stock_balances b
		                   JOIN warehouse w ON w.id = b.warehouse_id AND w.enterprise_id = b.enterprise_id
		                  WHERE b.enterprise_id = $1 AND b.item_code = i.code AND w.disposition
		                    AND w.location::text NOT IN ('INSPECAO','REJEICAO','TRANSITO','TERCEIRO')), 0)
		  FROM items i WHERE i.code = $2 AND i.enterprise_id = $1`, id, item).Scan(&tipo, &fab, &compra, &fornecedor, &s.Livre)
	if err != nil {
		return s, fmt.Errorf("lendo o item %d para a promessa: %w", item, err)
	}
	s.Fabricado = tipo != nil && *tipo == 0
	if fab != nil {
		s.PrazoFabricacaoDias = *fab
	}
	if fornecedor != nil {
		s.PrazoCompraDias = *fornecedor
	} else if compra != nil {
		s.PrazoCompraDias = *compra
	}
	ler := func(sql string, args ...any) ([]service.Movimento, error) {
		rows, err := r.pool.Query(ctx, sql, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []service.Movimento
		for rows.Next() {
			var m service.Movimento
			if err := rows.Scan(&m.Data, &m.Quantidade, &m.Documento); err != nil {
				return nil, err
			}
			out = append(out, m)
		}
		return out, rows.Err()
	}
	hoje := time.Now().Format("2006-01-02")
	if s.Entradas, err = ler(`
		SELECT COALESCE(o.end_date, o.start_date, $3::date), o.planned_qty - COALESCE(o.produced_qty, 0) - COALESCE(o.scrapped_qty, 0), 'OF ' || o.order_number
		  FROM production_orders o
		 WHERE o.enterprise_id = $1 AND o.item_code = $2 AND o.status IN ('OPEN','IN_PROGRESS')
		   AND o.planned_qty - COALESCE(o.produced_qty, 0) - COALESCE(o.scrapped_qty, 0) > 0
		UNION ALL
		SELECT COALESCE(l.promised_date, l.delivery_date, $3::date), l.requested_qty - COALESCE(l.received_qty, 0) - COALESCE(l.cancelled_qty, 0), 'PC ' || p.code
		  FROM purchase_order_items l JOIN purchase_orders p ON p.code = l.purchase_order_code AND p.enterprise_code = $4
		 WHERE l.item_code = $2 AND p.status IN ('REQUESTED','APPROVED','PARTIAL') AND COALESCE(l.status,'') NOT IN ('CANCELLED','RECEIVED')
		   AND l.requested_qty - COALESCE(l.received_qty, 0) - COALESCE(l.cancelled_qty, 0) > 0`, id, item, hoje, code); err != nil {
		return s, fmt.Errorf("lendo as entradas previstas do item %d: %w", item, err)
	}
	// Compromissos: o pendente das linhas de pedido abertas que ainda não tem reserva.
	if s.Compromissos, err = ler(`
		SELECT COALESCE(l.delivery_date, $3::date),
		       l.requested_qty - COALESCE(l.attended_qty, 0) - COALESCE(l.cancelled_qty, 0)
		         - COALESCE((SELECT SUM(r.quantity) FROM stock_reservations r WHERE r.enterprise_id = $1 AND r.status = 'ACTIVE'
		                      AND r.reference_type = 'SALES_ORDER' AND r.reference_code = so.code AND r.reference_item_code = l.code), 0),
		       'PV ' || so.order_number
		  FROM sales_order_items l JOIN sales_orders so ON so.code = l.sales_order_code AND so.enterprise_code = $4
		 WHERE l.item_code = $2 AND so.status IN ('P','A') AND COALESCE(l.is_active, TRUE) AND COALESCE(l.status,'') <> 'CANCELLED'
		   AND l.requested_qty - COALESCE(l.attended_qty, 0) - COALESCE(l.cancelled_qty, 0)
		         - COALESCE((SELECT SUM(r.quantity) FROM stock_reservations r WHERE r.enterprise_id = $1 AND r.status = 'ACTIVE'
		                      AND r.reference_type = 'SALES_ORDER' AND r.reference_code = so.code AND r.reference_item_code = l.code), 0) > 0`, id, item, hoje, code); err != nil {
		return s, fmt.Errorf("lendo os compromissos do item %d: %w", item, err)
	}
	if s.Fabricado {
		// Fila: horas abertas nos centros do roteiro padrão ÷ horas/dia das máquinas deles.
		var fila decimal.Decimal
		if err := r.pool.QueryRow(ctx, `
			WITH centros AS (
			    SELECT DISTINCT ro.work_center_id c FROM manufacturing_routes mr
			      JOIN route_operations ro ON ro.route_id = mr.id AND COALESCE(ro.is_active, TRUE)
			     WHERE mr.enterprise_id = $1 AND mr.item_code = $2 AND mr.is_standard AND ro.work_center_id IS NOT NULL),
			carga AS (
			    SELECT COALESCE(SUM(GREATEST(COALESCE(op.planned_hours,0) + COALESCE(op.setup_hours,0) - COALESCE(op.actual_hours,0), 0)), 0) h
			      FROM production_order_operations op
			     WHERE op.enterprise_id = $1 AND op.work_center_id IN (SELECT c FROM centros) AND op.status NOT IN ('DONE','SKIPPED')),
			capacidade AS (
			    SELECT COALESCE(SUM(COALESCE(m.available_hours_per_day, 8)), 0) h FROM machines m
			      JOIN machine_types t ON t.code = m.machine_type_code AND t.enterprise_id = m.enterprise_id
			     WHERE m.enterprise_id = $1 AND m.is_active AND t.id IN (SELECT c FROM centros))
			SELECT CASE WHEN capacidade.h > 0 THEN CEIL(carga.h / capacidade.h) ELSE 0 END FROM carga, capacidade`, id, item).Scan(&fila); err != nil {
			return s, fmt.Errorf("lendo a fila de capacidade do item %d: %w", item, err)
		}
		s.FilaDias = int(fila.IntPart())
	}
	return s, nil
}

// ComponentesParaPromessa: a estrutura ativa do item (sem co-produto nem
// material do cliente), na unidade de estoque.
func (r *Repository) ComponentesParaPromessa(ctx context.Context, item int64) ([]service.Componente, error) {
	id, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT s.child_code, SUM(COALESCE(s.quantity_stock_uom, s.quantity))
		  FROM item_structures s JOIN items i ON i.code = s.parent_code AND i.enterprise_id = $1
		 WHERE s.parent_code = $2 AND COALESCE(s.is_active, TRUE) AND NOT COALESCE(s.is_coproduct, FALSE) AND NOT COALESCE(s.customer_supplied, FALSE)
		   AND COALESCE(s.parent_mask, '') = ''
		 GROUP BY 1`, id, item)
	if err != nil {
		return nil, fmt.Errorf("lendo a estrutura do item %d: %w", item, err)
	}
	defer rows.Close()
	var out []service.Componente
	for rows.Next() {
		var c service.Componente
		if err := rows.Scan(&c.ItemCode, &c.QtdPor); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
