package mrp_report

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_report_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// Coverage lê, do último cálculo do plano, o que cada item precisa e recebe em
// cada data, mais o estoque atual (com a entrada/baixa simulada, se o plano é
// de um cenário) e as ordens planejadas ainda não firmadas. A
// conta de cobertura (dias, farol) é do caso de uso.
func (r *Repository) Coverage(ctx context.Context, filter mrp_report_uc.Filter) ([]mrp_report_uc.CoverageInput, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		WITH perfil AS (
		    SELECT p.item_code, p.need_date, SUM(p.demand) demand, SUM(p.orders_firm) firm, SUM(GREATEST(p.orders_planned, 0)) planned
		      FROM mrp_item_profiles p
		     WHERE p.enterprise_id = $1 AND p.plan_code = $2
		     GROUP BY p.item_code, p.need_date
		), pendentes AS (
		    SELECT s.item_code, MIN(s.start_date) primeiro_inicio, COUNT(*) quantas
		      FROM mrp_planned_suggestions s
		     WHERE s.enterprise_id = $1 AND s.plan_code = $2
		       AND NOT EXISTS (SELECT 1 FROM planned_orders po
		                        WHERE po.enterprise_id = $1 AND po.mrp_suggestion_code = s.code
		                          AND po.is_active AND po.status <> 'CANCELLED')
		     GROUP BY s.item_code
		), estoque AS (
		    -- Saldo real mais a entrada/baixa simulada, quando o plano é de um
		    -- cenário (plano real não tem ajuste e a segunda parte vem vazia).
		    SELECT item_code, SUM(saldo) saldo FROM (
		        SELECT b.item_code, b.quantity - b.reserved_qty AS saldo
		          FROM stock_balances b
		         WHERE b.enterprise_id = $1 AND ($5::bigint IS NULL OR b.warehouse_id = $5)
		        UNION ALL
		        SELECT c.item_code, c.quantity
		          FROM planning_scenario_changes c
		          JOIN planning_scenarios sc ON sc.id = c.scenario_id
		         WHERE sc.scenario_plan_code = $2 AND sc.enterprise_id = $1 AND c.kind = 'ESTOQUE'
		    ) x GROUP BY item_code
		)
		SELECT i.code, COALESCE(i.business_code::text, ''), i.name, COALESCE(i.warehouse_unit_of_measurement::text, ''),
		       COALESCE(i.engineering_type, 0), i.planner_employee_code, COALESCE(emp.name, ''),
		       COALESCE(i.planning_safety_stock, 0), COALESCE(e.saldo, 0),
		       pend.primeiro_inicio, COALESCE(pend.quantas, 0),
		       p.need_date, p.demand, p.firm, p.planned
		  FROM perfil p
		  JOIN items i ON i.code = p.item_code AND i.enterprise_id = $1
		  LEFT JOIN employees emp ON emp.code = i.planner_employee_code AND emp.enterprise_id = $1
		  LEFT JOIN estoque e ON e.item_code = i.code
		  LEFT JOIN pendentes pend ON pend.item_code = i.code
		 WHERE ($3::bigint IS NULL OR i.planner_employee_code = $3)
		   AND ($4 = '' OR $4 = 'TODOS' OR ($4 = 'FABRICADO' AND i.engineering_type = 0) OR ($4 = 'COMPRADO' AND i.engineering_type = 1)
		        OR ($4 IN ('DE_TERCEIRO', 'TERCEIRIZADO') AND i.engineering_type = 2))
		   AND ($6::bigint IS NULL OR i.code = $6)
		 ORDER BY i.code, p.need_date`,
		enterpriseID, filter.PlanCode, filter.Planner, filter.ItemType, filter.Warehouse, filter.ItemCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []mrp_report_uc.CoverageInput{}
	for rows.Next() {
		var in mrp_report_uc.CoverageInput
		var b mrp_report_uc.CoverageBucket
		var primeiro *time.Time
		var demand, firm, planned decimal.Decimal
		if err := rows.Scan(&in.ItemCode, &in.BusinessCode, &in.Description, &in.UOM,
			&in.EngineeringType, &in.PlannerCode, &in.PlannerName,
			&in.SafetyStock, &in.Stock, &primeiro, &in.PlannedOrders,
			&b.Date, &demand, &firm, &planned); err != nil {
			return nil, err
		}
		b.Demand, b.Firm, b.Planned = demand, firm, planned
		if n := len(out); n > 0 && out[n-1].ItemCode == in.ItemCode {
			out[n-1].Buckets = append(out[n-1].Buckets, b)
			continue
		}
		in.FirstPlannedStart = primeiro
		in.Buckets = []mrp_report_uc.CoverageBucket{b}
		out = append(out, in)
	}
	return out, rows.Err()
}
