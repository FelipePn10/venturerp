package customer_material

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/customer_material/entity"
	domrepo "github.com/FelipePn10/panossoerp/internal/domain/customer_material/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// limiteRelatorio evita que um período em aberto traga o razão inteiro para a
// tela. O padrão cobre com folga um mês de beneficiamento.
const (
	limitePadraoRelatorio = 1000
	limiteMaximoRelatorio = 5000
)

// RelatorioDeMovimentos é a movimentação de material de terceiro, entrada e
// saída, com a nota do cliente e a nota de saída de cada movimento. SQL montado
// à mão pelo mesmo motivo do restante deste repositório: filtros opcionais.
//
// A ENTRADA não é linha do razão: o recebimento grava a linha da remessa com a
// quantidade conferida, e o saldo nasce dela. O relatório a apresenta como um
// movimento RECEIPT na data do recebimento — o ID é o da linha da remessa, então
// quem lista identifica a linha pelo par (tipo, ID). Remessa cancelada não entra:
// o material não chegou a ser recebido.
func (r *Repository) RelatorioDeMovimentos(ctx context.Context, f domrepo.FiltroMovimentos) ([]*entity.MovimentoDoRelatorio, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	conds := []string{"m.enterprise_id = $1"}
	args := []any{enterpriseID}
	adicionar := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if f.CustomerCode != nil {
		adicionar("r.customer_code = $%d", *f.CustomerCode)
	}
	if f.RemittanceID != nil {
		adicionar("r.id = $%d", *f.RemittanceID)
	}
	if f.De != nil {
		adicionar("m.created_at >= $%d::date", *f.De)
	}
	if f.Ate != nil {
		adicionar("m.created_at < ($%d::date + 1)", *f.Ate)
	}
	if len(f.Tipos) > 0 {
		adicionar("m.movement_type::text = ANY($%d)", f.Tipos)
	}
	if !f.IncluirEstornados {
		conds = append(conds, "m.reversed_at IS NULL")
	}
	if q := strings.ToLower(strings.TrimSpace(f.Q)); q != "" {
		args = append(args, "%"+q+"%")
		n := len(args)
		conds = append(conds, fmt.Sprintf(
			"(LOWER(i.description) LIKE $%d OR LOWER(i.customer_item_code) LIKE $%d OR LOWER(COALESCE(c.name,'')) LIKE $%d)", n, n, n))
	}
	limite := f.Limite
	if limite <= 0 {
		limite = limitePadraoRelatorio
	}
	if limite > limiteMaximoRelatorio {
		limite = limiteMaximoRelatorio
	}
	args = append(args, limite)

	rows, err := r.pool.Query(ctx, `
SELECT `+colunasMovimento+`,
       r.id, r.customer_code, COALESCE(c.name, ''), r.nfe_number, r.nfe_series,
       i.customer_item_code, i.description, i.ncm, i.uom,
       fe.numero_nf, fe.serie, fe.status, po.order_number
FROM (
    SELECT mm.id, mm.enterprise_id, mm.remittance_item_id, mm.movement_type::text AS movement_type,
           mm.quantity, mm.unit_value, mm.cfop, mm.production_order_id, mm.fiscal_exit_id,
           mm.scrap_destination::text AS scrap_destination, mm.reason, mm.idempotency_key,
           mm.created_by, mm.created_at, mm.reversed_at, mm.reversed_by, mm.reversal_reason
      FROM public.customer_material_movements mm
     WHERE mm.enterprise_id = $1
    UNION ALL
    SELECT ri.id, rr.enterprise_id, ri.id, 'RECEIPT',
           ri.qty_received, ri.unit_value, rr.cfop, NULL::bigint, NULL::bigint,
           NULL::text, ri.divergence_reason, '',
           rr.created_by, rr.received_at::timestamptz, NULL::timestamptz, NULL::uuid, NULL::text
      FROM public.customer_material_items ri
      JOIN public.customer_material_remittances rr ON rr.id = ri.remittance_id
     WHERE rr.enterprise_id = $1 AND rr.status::text <> 'CANCELADA'
) m
JOIN public.customer_material_items i ON i.id = m.remittance_item_id
JOIN public.customer_material_remittances r ON r.id = i.remittance_id
LEFT JOIN public.customers c ON c.code = r.customer_code AND c.enterprise_id = r.enterprise_id
LEFT JOIN public.fiscal_exits fe ON fe.id = m.fiscal_exit_id AND fe.enterprise_id = m.enterprise_id
LEFT JOIN public.production_orders po ON po.id = m.production_order_id AND po.enterprise_id = m.enterprise_id
WHERE `+strings.Join(conds, " AND ")+`
ORDER BY m.created_at DESC, m.id DESC
LIMIT $`+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, fmt.Errorf("relatório de movimentação de terceiros: %w", err)
	}
	defer rows.Close()

	var out []*entity.MovimentoDoRelatorio
	for rows.Next() {
		var l entity.MovimentoDoRelatorio
		var tipo string
		var destino *string
		m := &l.Movimento
		if err := rows.Scan(&m.ID, &m.EnterpriseID, &m.RemittanceItemID, &tipo,
			&m.Quantity, &m.UnitValue, &m.CFOP, &m.ProductionOrderID, &m.FiscalExitID,
			&destino, &m.Reason, &m.IdempotencyKey, &m.CreatedBy, &m.CreatedAt,
			&m.ReversedAt, &m.ReversedBy, &m.ReversalReason,
			&l.RemittanceID, &l.CustomerCode, &l.CustomerName, &l.NFeNumber, &l.NFeSeries,
			&l.CustomerItemCode, &l.Description, &l.NCM, &l.UOM,
			&l.NotaSaidaNumero, &l.NotaSaidaSerie, &l.NotaSaidaStatus, &l.ProductionOrderNumber); err != nil {
			return nil, err
		}
		m.MovementType = entity.TipoMovimento(tipo)
		if destino != nil && *destino != "" {
			d := entity.DestinoSucata(*destino)
			m.ScrapDestination = &d
		}
		out = append(out, &l)
	}
	return out, rows.Err()
}

// HorasDeBeneficiamento soma as horas das OFs de beneficiamento — a OF cujo
// item tem, na estrutura, componente fornecido pelo cliente (migração 000453).
//
// Duas fontes, mostradas lado a lado:
//   - apontamentos (início/fim) — a mesma base do custo real da OF;
//   - lançamentos nas operações do roteiro, lidos da trilha de execução
//     (migração 000359), que guarda o delta de horas com o momento.
//
// O cliente vem do pedido de venda da ordem planejada que gerou a OF: pelo vínculo
// gravado ao firmar a sugestão do MRP (sales_order_planned_orders) ou, na ordem
// criada direto da demanda, por planned_orders.sales_order_code →
// sales_order_demands. Se a ordem atende mais de um pedido, vale o de maior
// quantidade. O pedido é do comercial: a junção usa enterprise_code (tenant.Code).
func (r *Repository) HorasDeBeneficiamento(ctx context.Context, f domrepo.FiltroHoras) ([]*entity.HorasDeBeneficiamento, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	enterpriseCode, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	conds := []string{"o.enterprise_id = $1"}
	args := []any{enterpriseID, enterpriseCode, f.De, f.Ate}
	adicionar := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if f.CustomerCode != nil {
		adicionar("so.customer_code = $%d", *f.CustomerCode)
	}
	if f.SalesOrderCode != nil {
		adicionar("so.code = $%d", *f.SalesOrderCode)
	}

	rows, err := r.pool.Query(ctx, `
WITH ap AS (
    SELECT a.production_order_id,
           SUM(CASE WHEN a.start_time IS NOT NULL AND a.end_time IS NOT NULL THEN
                   EXTRACT(EPOCH FROM (a.end_time - a.start_time
                       + CASE WHEN a.end_time < a.start_time THEN INTERVAL '24 hours' ELSE INTERVAL '0' END)) / 3600.0
               ELSE 0 END)::numeric(14,4) AS horas,
           COUNT(*)::int AS qtd,
           MIN(a.appointment_date)::timestamptz AS primeiro,
           MAX(a.appointment_date)::timestamptz AS ultimo
      FROM public.production_appointments a
     WHERE ($3::date IS NULL OR a.appointment_date >= $3::date)
       AND ($4::date IS NULL OR a.appointment_date <= $4::date)
     GROUP BY a.production_order_id
), op AS (
    SELECT e.production_order_id,
           SUM(e.actual_hours_delta)::numeric(14,4) AS horas,
           MIN(e.occurred_at) AS primeiro,
           MAX(e.occurred_at) AS ultimo
      FROM public.production_operation_execution_events e
     WHERE e.enterprise_id = $1 AND e.actual_hours_delta <> 0
       AND ($3::date IS NULL OR e.occurred_at >= $3::date)
       AND ($4::date IS NULL OR e.occurred_at < ($4::date + 1))
     GROUP BY e.production_order_id
)
SELECT o.id, o.order_number, o.status::text, o.item_code,
       COALESCE(i.business_code, o.item_code::text), COALESCE(NULLIF(i.pdm_description_technique, ''), i.name, ''),
       o.planned_qty, o.produced_qty,
       so.customer_code, COALESCE(c.name, ''), so.code, so.order_number,
       COALESCE(ap.horas, 0), COALESCE(ap.qtd, 0), COALESCE(op.horas, 0),
       LEAST(ap.primeiro, op.primeiro), GREATEST(ap.ultimo, op.ultimo)
FROM public.production_orders o
JOIN public.items i ON i.code = o.item_code AND i.enterprise_id = o.enterprise_id
LEFT JOIN public.planned_orders po ON po.id = o.planned_order_id AND po.enterprise_id = o.enterprise_id
LEFT JOIN LATERAL (
    SELECT origem.sales_order_code FROM (
        SELECT o.sales_order_code, 0::numeric AS quantity, 0 AS prioridade
         WHERE o.sales_order_code IS NOT NULL
        UNION ALL
        SELECT soi.sales_order_code, v.quantity, 1 AS prioridade
          FROM public.sales_order_planned_orders v
          JOIN public.sales_order_items soi ON soi.code = v.sales_order_item_code
         WHERE v.planned_order_id = po.id AND v.enterprise_id = o.enterprise_id
        UNION ALL
        SELECT demand.sales_order_code, demand.quantity, 2
          FROM public.sales_order_demands demand
         WHERE demand.code = po.sales_order_code AND demand.enterprise_id = o.enterprise_id
    ) origem
    ORDER BY origem.prioridade, origem.quantity DESC, origem.sales_order_code
    LIMIT 1
) ligacao ON TRUE
LEFT JOIN public.sales_orders so ON so.code = ligacao.sales_order_code AND so.enterprise_code = $2
LEFT JOIN public.customers c ON c.code = so.customer_code AND c.enterprise_id = o.enterprise_id
LEFT JOIN ap ON ap.production_order_id = o.id
LEFT JOIN op ON op.production_order_id = o.id
WHERE `+strings.Join(conds, " AND ")+`
  AND o.is_active
  AND EXISTS (SELECT 1 FROM public.item_structures s
               WHERE s.parent_code = o.item_code AND s.is_active AND s.customer_supplied)
  AND (ap.production_order_id IS NOT NULL OR op.production_order_id IS NOT NULL)
ORDER BY COALESCE(c.name, '~'), o.order_number`, args...)
	if err != nil {
		return nil, fmt.Errorf("relatório de horas de beneficiamento: %w", err)
	}
	defer rows.Close()

	var out []*entity.HorasDeBeneficiamento
	for rows.Next() {
		var h entity.HorasDeBeneficiamento
		if err := rows.Scan(&h.ProductionOrderID, &h.OrderNumber, &h.Status, &h.ItemCode,
			&h.ItemBusinessCode, &h.ItemDescription, &h.PlannedQty, &h.ProducedQty,
			&h.CustomerCode, &h.CustomerName, &h.SalesOrderCode, &h.SalesOrderNumber,
			&h.HorasApontadas, &h.Apontamentos, &h.HorasDeOperacao,
			&h.PrimeiroRegistro, &h.UltimoRegistro); err != nil {
			return nil, err
		}
		out = append(out, &h)
	}
	return out, rows.Err()
}
