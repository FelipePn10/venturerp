package mrp_report

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/mrp_report_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

type Repository struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) Profile(ctx context.Context, filter mrp_report_uc.Filter) ([]mrp_report_uc.ReportRow, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT profile.item_code,'',profile.need_date,profile.demand,
		profile.orders_planned,profile.orders_firm,CASE WHEN UPPER($8)='CURRENT' THEN COALESCE(current_stock.quantity,0) ELSE 0 END,profile.stock_projected,
		item.planner_employee_code,
		COALESCE((SELECT classification.code FROM item_classification_assignments assignment JOIN item_classifications classification ON classification.id=assignment.classification_id
			WHERE assignment.enterprise_id=$1 AND assignment.item_code=profile.item_code ORDER BY classification.level DESC,classification.code LIMIT 1),''),
		CASE WHEN $12 THEN COALESCE((SELECT array_agg(DISTINCT demand.sales_order_code)
			FROM planned_orders planned JOIN sales_order_demands demand ON demand.code=planned.sales_order_code
			WHERE planned.enterprise_id=$1 AND planned.plan_code=profile.plan_code AND planned.item_code=profile.item_code),'{}'::bigint[]) ELSE '{}'::bigint[] END
		FROM mrp_item_profiles profile JOIN items item ON item.code=profile.item_code
		LEFT JOIN LATERAL(SELECT SUM(quantity-reserved_qty) quantity FROM stock_balances balance WHERE balance.enterprise_id=$1 AND balance.item_code=profile.item_code)current_stock ON TRUE
		WHERE profile.enterprise_id=$1
		AND ($2::bigint IS NULL OR profile.plan_code=$2) AND ($3::bigint IS NULL OR profile.item_code=$3)
		AND ($4::date IS NULL OR profile.need_date >= $4) AND ($5::date IS NULL OR profile.need_date <= $5)
		AND ($6::bigint IS NULL OR item.planner_employee_code=$6)
		AND ($7='' OR $7='TODOS' OR ($7='FABRICADO' AND item.engineering_type=0) OR ($7='COMPRADO' AND item.engineering_type=1) OR ($7 IN ('DE_TERCEIRO','TERCEIRIZADO') AND item.engineering_type=2))
		AND ($9::bigint IS NULL OR EXISTS (SELECT 1 FROM item_classification_assignments assignment
			JOIN item_classifications classification ON classification.id=assignment.classification_id
			JOIN item_classification_masks mask ON mask.id=classification.mask_id
			WHERE assignment.enterprise_id=$1 AND assignment.item_code=profile.item_code AND mask.code=$9
			AND classification.code LIKE REPLACE($10,'%%','%')))
		AND (NOT $11 OR EXISTS (SELECT 1 FROM mrp_exception_messages message WHERE message.enterprise_id=$1 AND message.plan_code=profile.plan_code AND message.item_code=profile.item_code))
		AND (NOT $13 OR (EXISTS(SELECT 1 FROM stock_snapshots snapshot WHERE snapshot.enterprise_id=$1 AND snapshot.item_code=profile.item_code AND snapshot.quantity<>0)
			AND NOT EXISTS(SELECT 1 FROM stock_movements movement WHERE movement.enterprise_id=$1 AND movement.item_code=profile.item_code)))
		ORDER BY profile.item_code,profile.need_date`, enterpriseID, filter.PlanCode, filter.ItemCode, filter.From, filter.To, filter.Planner, filter.ItemType, filter.Position, filter.ClassificationMaskCode, filter.ClassificationCode, filter.OnlyWithMessage, filter.IncludeSalesOrders, filter.OnlyStockWithoutReason)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []mrp_report_uc.ReportRow{}
	for rows.Next() {
		var row mrp_report_uc.ReportRow
		var date time.Time
		if err := rows.Scan(&row.ItemCode, &row.Mask, &date, &row.Demand, &row.PlannedSupply, &row.FirmSupply, &row.Stock, &row.ProjectedStock, &row.Planner, &row.Classification, &row.SalesOrders); err != nil {
			return nil, err
		}
		row.Date = &date
		row.Available = row.Stock.Add(row.PlannedSupply).Add(row.FirmSupply).Sub(row.Demand)
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if filter.Layout == "ANALITICO" {
		details, detailErr := r.pool.Query(ctx, `SELECT detail.item_code,detail.need_date,detail.detail_type,detail.source_code,detail.parent_item_code,detail.quantity
			FROM mrp_profile_details detail WHERE detail.enterprise_id=$1 AND ($2::bigint IS NULL OR detail.plan_code=$2)
			AND ($3::bigint IS NULL OR detail.item_code=$3) AND ($4::date IS NULL OR detail.need_date >= $4) AND ($5::date IS NULL OR detail.need_date <= $5)
			ORDER BY detail.item_code,detail.need_date,detail.id`, enterpriseID, filter.PlanCode, filter.ItemCode, filter.From, filter.To)
		if detailErr != nil {
			return nil, detailErr
		}
		defer details.Close()
		for details.Next() {
			var detail mrp_report_uc.ReportRow
			var date time.Time
			if err := details.Scan(&detail.ItemCode, &date, &detail.SourceType, &detail.SourceCode, &detail.ParentItemCode, &detail.Demand); err != nil {
				return nil, err
			}
			detail.Date = &date
			detail.RowType = "DETALHE"
			result = append(result, detail)
		}
		if err := details.Err(); err != nil {
			return nil, err
		}
	}
	if filter.IncludeDrawings {
		if err := r.attachDrawings(ctx, enterpriseID, result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *Repository) Availability(ctx context.Context, filter mrp_report_uc.Filter) ([]mrp_report_uc.ReportRow, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `WITH RECURSIVE roots AS (
		SELECT line.item_code,line.mask,GREATEST(line.requested_qty-line.attended_qty-line.cancelled_qty,0)::numeric quantity,header.order_number sales_order,0 level
		FROM sales_order_items line JOIN sales_orders header ON header.code=line.sales_order_code JOIN enterprise company ON company.code=header.enterprise_code
		WHERE company.id=$1 AND COALESCE(cardinality($2::bigint[]),0)>0 AND header.order_number=ANY($2::bigint[]) AND header.is_active AND NOT header.is_blocked
		AND line.is_active AND line.status IN ('OPEN','PARTIAL')
		UNION ALL SELECT $3::bigint,''::varchar,$4::numeric,NULL::bigint,0 WHERE COALESCE(cardinality($2::bigint[]),0)=0 AND $3::bigint IS NOT NULL AND $4::numeric>0
	), needs AS (
		SELECT item_code,mask,quantity,sales_order,level,ARRAY[item_code] path FROM roots
		UNION ALL SELECT structure.child_code,''::varchar,needs.quantity*structure.quantity*(1+structure.loss_percentage/100),needs.sales_order,needs.level+1,needs.path||structure.child_code
		FROM needs JOIN item_structures structure ON structure.parent_code=needs.item_code AND structure.is_active
		WHERE NOT structure.child_code=ANY(needs.path)
	), grouped AS (
		SELECT item_code,MAX(mask) mask,SUM(quantity) demand,array_agg(DISTINCT sales_order) FILTER(WHERE sales_order IS NOT NULL) sales_orders,level FROM needs GROUP BY item_code,level
	), stock AS (
		SELECT item_code,SUM(quantity-reserved_qty) stock FROM stock_balances WHERE enterprise_id=$1 AND ($5::bigint IS NULL OR warehouse_id=$5) GROUP BY item_code
	), supply AS (
		SELECT item_code,SUM(orders_planned) planned,SUM(orders_firm) firm FROM mrp_item_profiles WHERE enterprise_id=$1 AND ($6::bigint IS NULL OR plan_code=$6) GROUP BY item_code)
		SELECT grouped.item_code,grouped.mask,grouped.demand,COALESCE(supply.planned,0),COALESCE(supply.firm,0),COALESCE(stock.stock,0),COALESCE(grouped.sales_orders,'{}'::bigint[]),grouped.level
		FROM grouped JOIN items item ON item.code=grouped.item_code LEFT JOIN stock USING(item_code) LEFT JOIN supply USING(item_code)
		WHERE ($7::bigint IS NULL OR item.planner_employee_code=$7)
		AND ($8='' OR $8='TODOS' OR ($8='FABRICADO' AND item.engineering_type=0) OR ($8='COMPRADO' AND item.engineering_type=1) OR ($8 IN ('DE_TERCEIRO','TERCEIRIZADO') AND item.engineering_type=2))
		AND ($9::bigint IS NULL OR EXISTS (SELECT 1 FROM item_classification_assignments assignment
			JOIN item_classifications classification ON classification.id=assignment.classification_id
			JOIN item_classification_masks mask ON mask.id=classification.mask_id
			WHERE assignment.enterprise_id=$1 AND assignment.item_code=grouped.item_code AND mask.code=$9
			AND classification.code LIKE REPLACE($10,'%%','%')))
		AND ($11='' OR $11='AMBOS' OR ($11='NECESSIDADES' AND grouped.level>0) OR ($11='ITENS_PEDIDO' AND grouped.level=0))
		ORDER BY grouped.level,grouped.item_code`, enterpriseID, filter.SalesOrderCodes, filter.ItemCode, filter.Quantity, filter.Warehouse, filter.PlanCode, filter.Planner, filter.ItemType, filter.ClassificationMaskCode, filter.ClassificationCode, filter.Layout)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []mrp_report_uc.ReportRow{}
	for rows.Next() {
		var row mrp_report_uc.ReportRow
		var level int
		if err := rows.Scan(&row.ItemCode, &row.Mask, &row.Demand, &row.PlannedSupply, &row.FirmSupply, &row.Stock, &row.SalesOrders, &level); err != nil {
			return nil, err
		}
		row.Available = row.Stock.Add(row.PlannedSupply).Add(row.FirmSupply).Sub(row.Demand)
		row.Level = &level
		if level == 0 {
			row.RowType = "ITEM_PEDIDO"
		} else {
			row.RowType = "NECESSIDADE"
		}
		if !filter.OnlyAvailable || row.Available.IsPositive() {
			result = append(result, row)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if filter.IncludeDrawings {
		if err := r.attachDrawings(ctx, enterpriseID, result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *Repository) attachDrawings(ctx context.Context, enterpriseID int64, result []mrp_report_uc.ReportRow) error {
	itemSet := make(map[int64]struct{})
	for _, row := range result {
		itemSet[row.ItemCode] = struct{}{}
	}
	items := make([]int64, 0, len(itemSet))
	for item := range itemSet {
		items = append(items, item)
	}
	if len(items) == 0 {
		return nil
	}
	rows, err := r.pool.Query(ctx, `SELECT drawing.item_code,LEFT(drawing.code,20)||drawing.digit||drawing.format||revision.revision
		FROM drawings drawing JOIN drawing_revisions revision ON revision.drawing_id=drawing.id
		WHERE drawing.enterprise_id=$1 AND drawing.item_code=ANY($2::bigint[]) AND drawing.is_active AND revision.is_current
		AND (revision.start_date IS NULL OR revision.start_date<=CURRENT_DATE) AND (revision.end_date IS NULL OR revision.end_date>=CURRENT_DATE)
		ORDER BY drawing.item_code,drawing.code,drawing.digit,drawing.format`, enterpriseID, items)
	if err != nil {
		return err
	}
	defer rows.Close()
	byItem := make(map[int64][]string)
	for rows.Next() {
		var item int64
		var code string
		if err := rows.Scan(&item, &code); err != nil {
			return err
		}
		byItem[item] = append(byItem[item], code)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range result {
		result[i].DrawingCodes = byItem[result[i].ItemCode]
	}
	return nil
}

func (r *Repository) GroupedNeeds(ctx context.Context, filter mrp_report_uc.Filter) ([]mrp_report_uc.ReportRow, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT profile.item_code,profile.need_date,SUM(profile.demand),SUM(profile.orders_planned),SUM(profile.orders_firm),
		COALESCE((SELECT SUM(balance.quantity-balance.reserved_qty) FROM stock_balances balance WHERE balance.enterprise_id=$1 AND balance.item_code=profile.item_code),0),item.planner_employee_code,
		COALESCE((SELECT classification.code FROM item_classification_assignments assignment JOIN item_classifications classification ON classification.id=assignment.classification_id WHERE assignment.enterprise_id=$1 AND assignment.item_code=profile.item_code ORDER BY classification.level DESC,classification.code LIMIT 1),''),
		CASE WHEN $10 THEN COALESCE((SELECT array_agg(DISTINCT demand.sales_order_code) FROM planned_orders planned JOIN sales_order_demands demand ON demand.code=planned.sales_order_code WHERE planned.enterprise_id=$1 AND planned.plan_code=profile.plan_code AND planned.item_code=profile.item_code),'{}'::bigint[]) ELSE '{}'::bigint[] END
		FROM mrp_item_profiles profile JOIN items item ON item.code=profile.item_code WHERE profile.enterprise_id=$1 AND ($2::bigint IS NULL OR profile.plan_code=$2)
		AND ($3::bigint IS NULL OR profile.item_code=$3) AND ($4::date IS NULL OR profile.need_date >= $4)
		AND ($5::date IS NULL OR profile.need_date <= $5) AND ($6::bigint IS NULL OR item.planner_employee_code=$6)
		AND ($7='' OR $7='TODOS' OR ($7='FABRICADO' AND item.engineering_type=0) OR ($7='COMPRADO' AND item.engineering_type=1) OR ($7 IN ('DE_TERCEIRO','TERCEIRIZADO') AND item.engineering_type=2))
		AND ($8::bigint IS NULL OR EXISTS (SELECT 1 FROM item_classification_assignments assignment
			JOIN item_classifications classification ON classification.id=assignment.classification_id
			JOIN item_classification_masks mask ON mask.id=classification.mask_id
			WHERE assignment.enterprise_id=$1 AND assignment.item_code=profile.item_code AND mask.code=$8
			AND classification.code LIKE REPLACE($9,'%%','%')))
		GROUP BY profile.item_code,profile.plan_code,profile.need_date,item.planner_employee_code ORDER BY profile.need_date,profile.item_code`, enterpriseID, filter.PlanCode, filter.ItemCode, filter.From, filter.To, filter.Planner, filter.ItemType, filter.ClassificationMaskCode, filter.ClassificationCode, filter.IncludeSalesOrders)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []mrp_report_uc.ReportRow{}
	for rows.Next() {
		var row mrp_report_uc.ReportRow
		var date time.Time
		if err := rows.Scan(&row.ItemCode, &date, &row.Demand, &row.PlannedSupply, &row.FirmSupply, &row.Stock, &row.Planner, &row.Classification, &row.SalesOrders); err != nil {
			return nil, err
		}
		row.Date = &date
		row.Required = row.Demand.Sub(row.PlannedSupply).Sub(row.FirmSupply)
		if row.Required.IsNegative() {
			row.Required = decimal.Zero
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if filter.IncludeDrawings {
		if err := r.attachDrawings(ctx, enterpriseID, result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (r *Repository) Explosion(ctx context.Context, itemCode int64, quantity decimal.Decimal, at *time.Time, filter mrp_report_uc.Filter) ([]mrp_report_uc.ReportRow, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `WITH RECURSIVE roots AS (
		SELECT $1::bigint item_code,$2::numeric quantity WHERE COALESCE(cardinality($10::bigint[]),0)=0 AND COALESCE(cardinality($11::bigint[]),0)=0
		UNION ALL SELECT production.item_code,GREATEST(production.planned_qty-production.produced_qty,0) FROM production_orders production
		WHERE production.enterprise_id=$4 AND production.order_number=ANY($10::bigint[]) AND production.status IN ('OPEN','IN_PROGRESS')
		UNION ALL SELECT shipment_item.item_code,shipment_item.quantity FROM shipment_loads load JOIN shipment_load_shipments link ON link.load_id=load.id
		JOIN shipments shipment ON shipment.id=link.shipment_id JOIN shipment_items shipment_item ON shipment_item.shipment_id=shipment.id
		JOIN sales_orders sale ON sale.code=shipment.sales_order_code JOIN enterprise company ON company.code=sale.enterprise_code
		WHERE company.id=$4 AND load.code=ANY($11::bigint[]) AND load.status<>'CANCELLED'
	), explosion AS (
		SELECT structure.parent_code,structure.child_code,1 level,(roots.quantity*structure.quantity*(1+structure.loss_percentage/100)) quantity,ARRAY[structure.parent_code,structure.child_code] path
		FROM roots JOIN item_structures structure ON structure.parent_code=roots.item_code WHERE structure.is_active
		AND ($3::date IS NULL OR structure.start_date IS NULL OR structure.start_date <= $3)
		AND ($3::date IS NULL OR structure.end_date IS NULL OR structure.end_date >= $3)
		UNION ALL SELECT structure.parent_code,structure.child_code,explosion.level+1,
		explosion.quantity*structure.quantity*(1+structure.loss_percentage/100),explosion.path||structure.child_code
		FROM explosion JOIN item_structures structure ON structure.parent_code=explosion.child_code AND structure.is_active
		WHERE NOT structure.child_code=ANY(explosion.path))
		SELECT explosion.parent_code,explosion.child_code,explosion.level,SUM(explosion.quantity),
		COALESCE(stock.quantity,0),COALESCE(purchase.quantity,0),CASE WHEN $13='RESUMIDA' THEN COALESCE(item.complement,'') ELSE item.pdm_description_technique END,item.warehouse_unit_of_measurement::text,COALESCE(cost.total_cost,0)
		FROM explosion JOIN items item ON item.code=explosion.child_code
		LEFT JOIN LATERAL (SELECT SUM(balance.quantity-balance.reserved_qty) quantity FROM stock_balances balance WHERE balance.enterprise_id=$4 AND balance.item_code=explosion.child_code AND ($5::bigint IS NULL OR balance.warehouse_id=$5) AND (NOT $12 OR balance.warehouse_id=item.warehouse_code)) stock ON TRUE
		LEFT JOIN LATERAL (SELECT SUM(line.requested_qty-line.received_qty-line.cancelled_qty) quantity FROM purchase_order_items line JOIN purchase_orders header ON header.code=line.purchase_order_code JOIN enterprise company ON company.code=header.enterprise_code WHERE company.id=$4 AND line.item_code=explosion.child_code AND line.is_active AND line.status IN ('OPEN','PARTIAL')) purchase ON TRUE
		LEFT JOIN item_standard_costs cost ON cost.item_code=explosion.child_code AND cost.mask=''
		WHERE ($6='' OR ($6='FABRICADO' AND item.engineering_type=0) OR ($6='COMPRADO' AND item.engineering_type=1) OR ($6 IN ('DE_TERCEIRO','TERCEIRIZADO') AND item.engineering_type=2))
		AND ($7::bigint IS NULL OR EXISTS (SELECT 1 FROM item_classification_assignments assignment JOIN item_classifications classification ON classification.id=assignment.classification_id JOIN item_classification_masks mask ON mask.id=classification.mask_id WHERE assignment.enterprise_id=$4 AND assignment.item_code=explosion.child_code AND mask.code=$7 AND classification.code LIKE REPLACE($8,'%%','%')))
		AND ($9 <> 'FILHOS_IMEDIATOS' OR explosion.level=1)
		GROUP BY explosion.parent_code,explosion.child_code,explosion.level,stock.quantity,purchase.quantity,item.pdm_description_technique,item.complement,item.warehouse_unit_of_measurement,cost.total_cost ORDER BY explosion.level,explosion.parent_code,explosion.child_code`, itemCode, quantity, at, enterpriseID, filter.Warehouse, filter.ItemType, filter.ClassificationMaskCode, filter.ClassificationCode, filter.ListMode, filter.ProductionOrderCodes, filter.LoadCodes, filter.ConsiderItemWarehouses, filter.DescriptionType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []mrp_report_uc.ReportRow{}
	for rows.Next() {
		var row mrp_report_uc.ReportRow
		var level int
		if err := rows.Scan(&row.ParentItemCode, &row.ItemCode, &level, &row.Required, &row.Stock, &row.PurchaseSupply, &row.Description, &row.UOM, &row.Cost); err != nil {
			return nil, err
		}
		row.Level = &level
		row.Cost = row.Cost.Mul(row.Required)
		row.Available = row.Stock.Add(row.PurchaseSupply)
		switch filter.ExplosionOption {
		case "CUSTO":
			row.Stock, row.PurchaseSupply, row.Available = decimal.Zero, decimal.Zero, decimal.Zero
		case "SALDO", "SALDO_DEM":
			row.Cost = decimal.Zero
			row.ProjectedStock = row.Available.Sub(row.Required)
		default:
			row.Stock, row.PurchaseSupply, row.Available, row.Cost = decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *Repository) ReorderPoint(ctx context.Context, filter mrp_report_uc.Filter) ([]mrp_report_uc.ReportRow, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `WITH RECURSIVE sales_needs AS (
		SELECT line.item_code,GREATEST(line.requested_qty-line.attended_qty-line.cancelled_qty,0)::numeric quantity,ARRAY[line.item_code] path
		FROM sales_order_items line JOIN sales_orders header ON header.code=line.sales_order_code JOIN enterprise company ON company.code=header.enterprise_code
		WHERE company.id=$1 AND header.is_active AND header.status NOT IN ('F','CANCELLED') AND line.is_active AND line.status IN ('OPEN','PARTIAL')
		AND ($12='LIBERADOS_E_BLOQUEADOS' OR NOT header.is_blocked) AND ($9::date IS NULL OR COALESCE(line.delivery_date,header.delivery_date)>=$9) AND ($10::date IS NULL OR COALESCE(line.delivery_date,header.delivery_date)<=$10)
		UNION ALL SELECT structure.child_code,needs.quantity*structure.quantity*(1+structure.loss_percentage/100),needs.path||structure.child_code
		FROM sales_needs needs JOIN item_structures structure ON structure.parent_code=needs.item_code AND structure.is_active WHERE NOT structure.child_code=ANY(needs.path)
	)
	SELECT balance.item_code,COALESCE(SUM(CASE WHEN $8 THEN balance.quantity-balance.reserved_qty ELSE balance.quantity END),0),
		COALESCE(MAX(balance.safety_stock),0),COALESCE(MAX(balance.maximum_stock),0),COALESCE(avg.avg_monthly_consumption,0),
		COALESCE((SELECT SUM(material.quantity-material.attended_quantity) FROM production_order_materials material JOIN production_orders production ON production.id=material.production_order_id WHERE material.enterprise_id=$1 AND material.item_code=balance.item_code AND material.material_kind='DEMAND' AND production.status IN ('OPEN','IN_PROGRESS') AND ($9::date IS NULL OR production.end_date >= $9) AND ($10::date IS NULL OR production.end_date <= $10)),0)
		+COALESCE((SELECT SUM(need.quantity) FROM sales_needs need WHERE need.item_code=balance.item_code),0),
		COALESCE((SELECT SUM(line.requested_qty-line.received_qty-line.cancelled_qty) FROM purchase_order_items line JOIN purchase_orders header ON header.code=line.purchase_order_code JOIN enterprise company ON company.code=header.enterprise_code WHERE company.id=$1 AND line.item_code=balance.item_code AND line.status IN ('OPEN','PARTIAL') AND line.is_active AND ($9::date IS NULL OR line.delivery_date >= $9) AND ($10::date IS NULL OR line.delivery_date <= $10)),0)
		FROM stock_balances balance JOIN items item ON item.code=balance.item_code LEFT JOIN item_consumption_averages avg ON avg.enterprise_id=balance.enterprise_id AND avg.item_code=balance.item_code
		WHERE balance.enterprise_id=$1 AND ($2::bigint IS NULL OR balance.item_code=$2) AND ($3::bigint IS NULL OR balance.warehouse_id=$3)
		AND ($4::bigint IS NULL OR item.planner_employee_code=$4)
		AND ($5='' OR $5='TODOS' OR ($5='FABRICADO' AND item.engineering_type=0) OR ($5='COMPRADO' AND item.engineering_type=1) OR ($5 IN ('DE_TERCEIRO','TERCEIRIZADO') AND item.engineering_type=2))
		AND ($6::bigint IS NULL OR EXISTS (SELECT 1 FROM item_classification_assignments assignment
			JOIN item_classifications classification ON classification.id=assignment.classification_id
			JOIN item_classification_masks mask ON mask.id=classification.mask_id
			WHERE assignment.enterprise_id=$1 AND assignment.item_code=balance.item_code AND mask.code=$6
			AND classification.code LIKE REPLACE($7,'%%','%')))
		AND ($11='' OR $11='TODOS' OR ($11='KANBAN' AND EXISTS(SELECT 1 FROM kanban_cards card WHERE card.enterprise_id=$1 AND card.item_code=balance.item_code AND card.status='ACTIVE')) OR ($11='REORDER_POINT' AND NOT EXISTS(SELECT 1 FROM kanban_cards card WHERE card.enterprise_id=$1 AND card.item_code=balance.item_code AND card.status='ACTIVE')))
		GROUP BY balance.item_code,avg.avg_monthly_consumption ORDER BY balance.item_code`, enterpriseID, filter.ItemCode, filter.Warehouse, filter.Planner, filter.ItemType, filter.ClassificationMaskCode, filter.ClassificationCode, filter.OnlyAvailable, filter.From, filter.To, filter.PlanningType, filter.OrderPosition)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []mrp_report_uc.ReportRow{}
	for rows.Next() {
		var row mrp_report_uc.ReportRow
		if err := rows.Scan(&row.ItemCode, &row.Available, &row.SafetyStock, &row.MaximumStock, &row.AverageMonthly, &row.Demand, &row.PurchaseSupply); err != nil {
			return nil, err
		}
		target := row.MaximumStock
		if target.IsZero() {
			target = row.SafetyStock.Add(row.AverageMonthly)
		}
		row.Required = target.Add(row.Demand).Sub(row.Available).Sub(row.PurchaseSupply)
		if row.Required.IsNegative() {
			row.Required = decimal.Zero
		}
		if row.Required.IsPositive() {
			result = append(result, row)
		}
	}
	return result, rows.Err()
}
