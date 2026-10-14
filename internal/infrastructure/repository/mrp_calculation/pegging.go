package mrp_calculation

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ReplacePegging troca o pegging do plano pelo do cálculo atual, numa
// transação: quem consulta no meio vê o anterior inteiro ou o novo inteiro.
func (r *MRPCalculationRepositorySQLC) ReplacePegging(ctx context.Context, planCode int64, records []entity.PeggingRecord) error {
	e, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	beginner, ok := r.db.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return fmt.Errorf("transação do pegging não configurada")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM mrp_pegging WHERE enterprise_id=$1 AND plan_code=$2`, e, planCode); err != nil {
		return fmt.Errorf("limpando o pegging do plano: %w", err)
	}
	if len(records) > 0 {
		n := len(records)
		items := make([]int64, n)
		dKinds := make([]string, n)
		dCodes := make([]*int64, n)
		needs := make([]time.Time, n)
		sKinds := make([]string, n)
		sCodes := make([]*int64, n)
		sDates := make([]*time.Time, n)
		qtds := make([]float64, n)
		for i, rec := range records {
			items[i], dKinds[i], dCodes[i], needs[i] = rec.ItemCode, rec.DemandKind, rec.DemandCode, rec.NeedDate
			sKinds[i], sCodes[i], sDates[i], qtds[i] = rec.SupplyKind, rec.SupplyCode, rec.SupplyDate, rec.Quantity
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO mrp_pegging (enterprise_id, plan_code, item_code, demand_kind, demand_code, need_date,
			                         supply_kind, supply_code, supply_date, quantity)
			SELECT $1, $2, u.item, u.dkind, u.dcode, u.need, u.skind, u.scode, u.sdate, u.qtd
			  FROM unnest($3::bigint[], $4::text[], $5::bigint[], $6::date[], $7::text[], $8::bigint[], $9::date[], $10::numeric[])
			       AS u(item, dkind, dcode, need, skind, scode, sdate, qtd)`,
			e, planCode, items, dKinds, dCodes, needs, sKinds, sCodes, sDates, qtds); err != nil {
			return fmt.Errorf("gravando o pegging do plano: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// PeggingRows devolve o pegging do último cálculo do plano.
func (r *MRPCalculationRepositorySQLC) PeggingRows(ctx context.Context, planCode int64) ([]entity.PeggingRecord, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT item_code, demand_kind, demand_code, need_date, supply_kind, supply_code, supply_date, quantity::float8
		  FROM mrp_pegging WHERE enterprise_id=$1 AND plan_code=$2 ORDER BY id`, e, planCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []entity.PeggingRecord{}
	for rows.Next() {
		var rec entity.PeggingRecord
		if err := rows.Scan(&rec.ItemCode, &rec.DemandKind, &rec.DemandCode, &rec.NeedDate,
			&rec.SupplyKind, &rec.SupplyCode, &rec.SupplyDate, &rec.Quantity); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// PeggingLabels traz o que identifica cada ponta do pegging para o usuário:
// número e tipo da sugestão/ordem, número do pedido e cliente, descrição do
// item. Tudo da empresa da sessão.
func (r *MRPCalculationRepositorySQLC) PeggingLabels(ctx context.Context, refs entity.PeggingRefs) (*entity.PeggingLabels, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	out := &entity.PeggingLabels{
		Suggestions: map[int64]entity.PeggingOrderLabel{}, FirmOrders: map[int64]entity.PeggingOrderLabel{},
		SalesOrderItems: map[int64]entity.PeggingSalesLabel{}, Items: map[int64]entity.PeggingItemLabel{},
	}
	if len(refs.Suggestions) > 0 {
		rows, err := r.db.Query(ctx, `SELECT s.code, s.order_number, s.order_type, s.item_code,
			       COALESCE(a.scheduled_start::date, s.start_date), s.need_date, s.quantity::float8, a.scheduled_end, a.scheduled_end::date
			  FROM mrp_planned_suggestions s
			  LEFT JOIN mrp_machine_allocations a ON a.suggestion_code = s.code AND a.enterprise_id = s.enterprise_id
			 WHERE s.enterprise_id=$1 AND s.code=ANY($2::bigint[])`, e, refs.Suggestions)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var code, item int64
			var l entity.PeggingOrderLabel
			if err := rows.Scan(&code, &l.OrderNumber, &l.OrderType, &item, &l.StartDate, &l.NeedDate, &l.Quantity, &l.EstimatedEnd, &l.EstimatedEndDay); err != nil {
				rows.Close()
				return nil, err
			}
			l.ItemCode = item
			out.Suggestions[code] = l
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	if len(refs.FirmOrders) > 0 {
		rows, err := r.db.Query(ctx, `SELECT code, order_number, order_type::text, item_code, start_date, need_date, quantity::float8
			FROM planned_orders WHERE enterprise_id=$1 AND code=ANY($2::bigint[])`, e, refs.FirmOrders)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var code int64
			var l entity.PeggingOrderLabel
			if err := rows.Scan(&code, &l.OrderNumber, &l.OrderType, &l.ItemCode, &l.StartDate, &l.NeedDate, &l.Quantity); err != nil {
				rows.Close()
				return nil, err
			}
			out.FirmOrders[code] = l
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	if len(refs.SalesOrderItems) > 0 {
		rows, err := r.db.Query(ctx, `
			SELECT soi.code, so.order_number, COALESCE(c.trade_name, c.name, '')
			  FROM sales_order_items soi
			  JOIN sales_orders so ON so.code = soi.sales_order_code
			  JOIN enterprise en ON en.code = so.enterprise_code
			  LEFT JOIN customers c ON c.code = so.customer_code AND c.enterprise_id = en.id
			 WHERE soi.code = ANY($2::bigint[]) AND en.id = $1`, e, refs.SalesOrderItems)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var code int64
			var l entity.PeggingSalesLabel
			if err := rows.Scan(&code, &l.OrderNumber, &l.Customer); err != nil {
				rows.Close()
				return nil, err
			}
			out.SalesOrderItems[code] = l
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	if len(refs.Items) > 0 {
		rows, err := r.db.Query(ctx, `SELECT code, COALESCE(business_code::text, ''), COALESCE(name, ''), COALESCE(warehouse_unit_of_measurement::text, '')
			FROM items WHERE enterprise_id=$1 AND code=ANY($2::bigint[])`, e, refs.Items)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var code int64
			var l entity.PeggingItemLabel
			if err := rows.Scan(&code, &l.BusinessCode, &l.Description, &l.UOM); err != nil {
				rows.Close()
				return nil, err
			}
			out.Items[code] = l
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// IsScenarioPlan diz se o plano é de um cenário de simulação — nada dele pode
// virar ordem real nem alterar ordem real.
func (r *MRPCalculationRepositorySQLC) IsScenarioPlan(ctx context.Context, planCode int64) (bool, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var cenario bool
	err = r.db.QueryRow(ctx, `SELECT COALESCE((SELECT is_scenario FROM production_plans WHERE code=$1 AND enterprise_id=$2), false)`, planCode, e).Scan(&cenario)
	return cenario, err
}

// SalesOrderPeggingPlans devolve os planos reais (não de cenário) cujo último
// cálculo vinculou itens do pedido de venda, com os itens vinculados.
func (r *MRPCalculationRepositorySQLC) SalesOrderPeggingPlans(ctx context.Context, orderCode int64) ([]entity.SalesOrderPlan, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.code, p.name, p.last_calculated_at, array_agg(DISTINCT soi.code)
		  FROM sales_orders so
		  JOIN enterprise en ON en.code = so.enterprise_code AND en.id = $2
		  JOIN sales_order_items soi ON soi.sales_order_code = so.code
		  JOIN mrp_pegging g ON g.enterprise_id = $2 AND g.demand_kind = 'PEDIDO_VENDA' AND g.demand_code = soi.code
		  JOIN production_plans p ON p.code = g.plan_code AND p.enterprise_id = $2 AND p.is_active AND NOT p.is_scenario
		 WHERE so.code = $1
		 GROUP BY p.code, p.name, p.last_calculated_at
		 ORDER BY p.last_calculated_at DESC NULLS LAST, p.code`, orderCode, e)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []entity.SalesOrderPlan{}
	for rows.Next() {
		var p entity.SalesOrderPlan
		if err := rows.Scan(&p.PlanCode, &p.PlanName, &p.CalculatedAt, &p.ItemCodes); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ItemBusinessCodes devolve o código de negócio dos itens da empresa da
// sessão (texto das mensagens de exceção). Item sem código de negócio fica de
// fora do mapa.
func (r *MRPCalculationRepositorySQLC) ItemBusinessCodes(ctx context.Context, codes []int64) (map[int64]string, error) {
	out := map[int64]string{}
	if len(codes) == 0 {
		return out, nil
	}
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT code, business_code::text FROM items
		WHERE enterprise_id=$1 AND code=ANY($2::bigint[]) AND COALESCE(business_code::text, '') <> ''`, e, codes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var code int64
		var bc string
		if err := rows.Scan(&code, &bc); err != nil {
			return nil, err
		}
		out[code] = bc
	}
	return out, rows.Err()
}
