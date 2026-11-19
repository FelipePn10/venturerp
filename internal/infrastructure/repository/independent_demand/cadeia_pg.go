package independent_demand

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
)

// vincularCadeia grava, pelo pegging da sugestão firmada, a cadeia de
// fabricação (ver .ai/tasks/cadeia-of-pai-filha.md):
//
//   - para cima: a necessidade que a sugestão cobre vem de uma ordem pai — a
//     sugestão do pai já firmada (SUGESTAO), a ordem planejada firme
//     (ORDEM_PLANEJADA) ou o empenho de uma OF (ORDEM_PRODUCAO);
//   - para baixo: componentes já firmados cuja necessidade vinha desta
//     sugestão (firmar de baixo para cima);
//   - o pedido de venda dos ancestrais desce para toda a cadeia abaixo.
//
// Idempotente: firmar de novo não duplica nada. O pegging só é lido aqui; o
// próximo cálculo pode apagá-lo sem perder a cadeia.
func (r *IndependentDemandRepositorySQLC) vincularCadeia(ctx context.Context, enterpriseID, suggestionCode, plannedOrderCode int64, por uuid.UUID) error {
	autor := pgutil.ToPgUUID(por)
	if _, err := r.db.Exec(ctx, `
		WITH pais AS (
		    SELECT CASE p.demand_kind
		             WHEN 'SUGESTAO' THEN (SELECT x.id FROM planned_orders x
		                                    WHERE x.enterprise_id = $1 AND x.mrp_suggestion_code = p.demand_code AND x.is_active)
		             WHEN 'ORDEM_PLANEJADA' THEN (SELECT x.id FROM planned_orders x
		                                           WHERE x.enterprise_id = $1 AND x.code = p.demand_code)
		             WHEN 'ORDEM_PRODUCAO' THEN (SELECT o.planned_order_id FROM production_orders o
		                                          WHERE o.enterprise_id = $1 AND o.id = p.demand_code)
		           END AS pai_planejada,
		           CASE WHEN p.demand_kind = 'ORDEM_PRODUCAO' THEN (SELECT o.id FROM production_orders o
		                                                             WHERE o.enterprise_id = $1 AND o.id = p.demand_code) END AS pai_of,
		           p.quantity
		      FROM mrp_pegging p
		     WHERE p.enterprise_id = $1 AND p.supply_kind = 'SUGESTAO' AND p.supply_code = $2
		       AND p.demand_kind IN ('SUGESTAO', 'ORDEM_PLANEJADA', 'ORDEM_PRODUCAO') AND p.quantity > 0
		)
		INSERT INTO production_chain_links (enterprise_id, child_planned_order_id, parent_planned_order_id, parent_production_order_id, quantity, created_by)
		SELECT $1, filho.id, pais.pai_planejada, pais.pai_of, SUM(pais.quantity), $4
		  FROM pais
		  JOIN planned_orders filho ON filho.code = $3 AND filho.enterprise_id = $1
		 WHERE (pais.pai_planejada IS NOT NULL OR pais.pai_of IS NOT NULL)
		   AND pais.pai_planejada IS DISTINCT FROM filho.id
		 GROUP BY filho.id, pais.pai_planejada, pais.pai_of
		ON CONFLICT DO NOTHING`, enterpriseID, suggestionCode, plannedOrderCode, autor); err != nil {
		return fmt.Errorf("ligando a sugestão %d às ordens pai: %w", suggestionCode, err)
	}
	if _, err := r.db.Exec(ctx, `
		INSERT INTO production_chain_links (enterprise_id, child_planned_order_id, parent_planned_order_id, quantity, created_by)
		SELECT $1, filho.id, pai.id, SUM(p.quantity), $4
		  FROM mrp_pegging p
		  JOIN planned_orders pai ON pai.code = $3 AND pai.enterprise_id = $1
		  JOIN planned_orders filho ON filho.enterprise_id = $1 AND filho.mrp_suggestion_code = p.supply_code AND filho.is_active
		 WHERE p.enterprise_id = $1 AND p.demand_kind = 'SUGESTAO' AND p.demand_code = $2
		   AND p.supply_kind = 'SUGESTAO' AND p.quantity > 0 AND filho.id <> pai.id
		 GROUP BY filho.id, pai.id
		ON CONFLICT DO NOTHING`, enterpriseID, suggestionCode, plannedOrderCode, autor); err != nil {
		return fmt.Errorf("ligando os componentes já firmados à sugestão %d: %w", suggestionCode, err)
	}
	// O pedido desce pela cadeia: cada ordem abaixo desta (inclusive ela)
	// recebe o pedido de cada ancestral. A quantidade é a da ordem do
	// componente — é vínculo de rastreio, não rateio do pedido.
	if _, err := r.db.Exec(ctx, `
		WITH RECURSIVE abaixo(id) AS (
		    SELECT po.id FROM planned_orders po WHERE po.code = $2 AND po.enterprise_id = $1
		    UNION
		    SELECT l.child_planned_order_id FROM production_chain_links l
		      JOIN abaixo a ON l.parent_planned_order_id = a.id
		     WHERE l.enterprise_id = $1
		), acima(descendente, ancestral) AS (
		    SELECT id, id FROM abaixo
		    UNION
		    SELECT s.descendente, l.parent_planned_order_id FROM acima s
		      JOIN production_chain_links l ON l.child_planned_order_id = s.ancestral
		     WHERE l.enterprise_id = $1 AND l.parent_planned_order_id IS NOT NULL
		)
		INSERT INTO sales_order_planned_orders (enterprise_id, sales_order_item_code, planned_order_id, quantity, created_by)
		SELECT DISTINCT $1::bigint, v.sales_order_item_code, s.descendente, po.quantity, $3::uuid
		  FROM acima s
		  JOIN sales_order_planned_orders v ON v.planned_order_id = s.ancestral AND v.enterprise_id = $1
		  JOIN planned_orders po ON po.id = s.descendente AND po.enterprise_id = $1 AND po.quantity > 0
		 WHERE s.descendente <> s.ancestral
		ON CONFLICT (enterprise_id, sales_order_item_code, planned_order_id) DO NOTHING`, enterpriseID, plannedOrderCode, autor); err != nil {
		return fmt.Errorf("levando o pedido de venda pela cadeia da ordem %d: %w", plannedOrderCode, err)
	}
	return nil
}
