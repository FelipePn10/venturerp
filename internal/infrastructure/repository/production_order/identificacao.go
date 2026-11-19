package production_order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// IdentificacaoDasOFs devolve a identificação de cada OF pedida, da empresa da
// sessão. A cadeia sobe pelos elos (planejada pai ou OF pai), como a CadeiaDaOF.
func (r *ProductionOrderRepositoryPGX) IdentificacaoDasOFs(ctx context.Context, ids []int64) (map[int64]entity.IdentificacaoOF, error) {
	out := map[int64]entity.IdentificacaoOF{}
	if len(ids) == 0 {
		return out, nil
	}
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE alvo AS (
		  SELECT o.id, o.planned_order_id FROM production_orders o WHERE o.enterprise_id = $1 AND o.id = ANY($2)
		), acima(origem, ofid, pl, nivel) AS (
		  SELECT a.id, a.id, a.planned_order_id, 0 FROM alvo a
		  UNION ALL
		  SELECT ac.origem, pai.ofid, pai.pl, ac.nivel + 1
		    FROM acima ac
		    JOIN LATERAL (
		      SELECT COALESCE(l.parent_production_order_id,
		                      (SELECT x.id FROM production_orders x WHERE x.enterprise_id = $1 AND x.planned_order_id = l.parent_planned_order_id
		                         AND x.is_active ORDER BY x.id DESC LIMIT 1)) AS ofid,
		             COALESCE(l.parent_planned_order_id, (SELECT y.planned_order_id FROM production_orders y WHERE y.id = l.parent_production_order_id AND y.enterprise_id = $1)) AS pl
		        FROM production_chain_links l
		       WHERE l.enterprise_id = $1 AND ac.pl IS NOT NULL AND l.child_planned_order_id = ac.pl
		       ORDER BY l.id LIMIT 1) pai ON TRUE
		   WHERE ac.nivel < 20
		), raiz AS (
		  SELECT DISTINCT ON (origem) origem, ofid, pl, nivel FROM acima ORDER BY origem, nivel DESC
		), pai AS (
		  SELECT origem, ofid, pl FROM acima WHERE nivel = 1
		)
		SELECT o.id, COALESCE(i.business_code::text, o.item_code::text), COALESCE(i.name, ''),
		       o.customer_code, COALESCE(c.name, ''), o.project_id, COALESCE(p.code, ''), COALESCE(p.name, ''),
		       COALESCE((SELECT array_agg(DISTINCT so.order_number ORDER BY so.order_number) FROM sales_orders so
		                  JOIN enterprise e ON e.code = so.enterprise_code AND e.id = $1
		                 WHERE so.code = ro.sales_order_code
		                    OR so.code IN (SELECT si.sales_order_code FROM sales_order_planned_orders sp JOIN sales_order_items si ON si.code = sp.sales_order_item_code
		                                    WHERE sp.enterprise_id = $1 AND sp.planned_order_id = rz.pl)), '{}'),
		       COALESCE((SELECT array_agg(DISTINCT d.code ORDER BY d.code) FROM independent_demand_orders v
		                  JOIN independent_demands d ON d.id = v.demand_id AND d.enterprise_id = v.enterprise_id
		                 WHERE v.enterprise_id = $1 AND (v.production_order_id = rz.ofid OR (rz.pl IS NOT NULL AND v.planned_order_id = rz.pl))), '{}'),
		       pa.ofid, CASE WHEN pa.ofid IS NULL THEN (SELECT po.order_number FROM planned_orders po WHERE po.id = pa.pl AND po.enterprise_id = $1) END,
		       COALESCE((SELECT COALESCE(ip.business_code::text, ip.code::text) FROM items ip
		                  WHERE ip.enterprise_id = $1 AND ip.code = COALESCE((SELECT x.item_code FROM production_orders x WHERE x.id = pa.ofid),
		                                                                    (SELECT y.item_code FROM planned_orders y WHERE y.id = pa.pl AND y.enterprise_id = $1))), ''),
		       (SELECT x.order_number FROM production_orders x WHERE x.id = pa.ofid AND x.enterprise_id = $1),
		       rz.ofid, ro.order_number,
		       COALESCE((SELECT COALESCE(ir.business_code::text, ir.code::text) FROM items ir
		                  WHERE ir.enterprise_id = $1 AND ir.code = COALESCE(ro.item_code, (SELECT y.item_code FROM planned_orders y WHERE y.id = rz.pl AND y.enterprise_id = $1))), ''),
		       rz.nivel,
		       (SELECT COUNT(DISTINCT f.id) FROM production_chain_links l
		          JOIN production_orders f ON f.enterprise_id = l.enterprise_id AND f.planned_order_id = l.child_planned_order_id AND f.is_active
		         WHERE l.enterprise_id = $1 AND (l.parent_production_order_id = o.id OR (l.parent_production_order_id IS NULL AND o.planned_order_id IS NOT NULL AND l.parent_planned_order_id = o.planned_order_id)))
		  FROM production_orders o
		  JOIN raiz rz ON rz.origem = o.id
		  LEFT JOIN pai pa ON pa.origem = o.id
		  LEFT JOIN production_orders ro ON ro.id = rz.ofid AND ro.enterprise_id = $1
		  LEFT JOIN items i ON i.code = o.item_code AND i.enterprise_id = o.enterprise_id
		  LEFT JOIN customers c ON c.enterprise_id = o.enterprise_id AND c.code = o.customer_code
		  LEFT JOIN projects p ON p.enterprise_id = o.enterprise_id AND p.id = o.project_id
		 WHERE o.enterprise_id = $1`, empresa, ids)
	if err != nil {
		return nil, fmt.Errorf("lendo a identificação das ordens: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var x entity.IdentificacaoOF
		var paiPlanejada *int64
		if err := rows.Scan(&x.OrderID, &x.ItemBusinessCode, &x.ItemName, &x.CustomerCode, &x.CustomerName, &x.ProjectID, &x.ProjectCode, &x.ProjectName,
			&x.SalesOrders, &x.Demands, &x.ParentOrderID, &paiPlanejada, &x.ParentItemBusiness, &x.ParentOrderNumber, &x.RootOrderID, &x.RootOrderNumber, &x.RootItemBusiness, &x.Level, &x.Children); err != nil {
			return nil, fmt.Errorf("lendo a identificação das ordens: %w", err)
		}
		x.ParentPlannedCode = paiPlanejada
		out[x.OrderID] = x
	}
	return out, rows.Err()
}

// DefinirOrigem grava cliente e projeto da OF (nil limpa). As filhas que
// herdaram o valor anterior acompanham (gatilho do banco).
func (r *ProductionOrderRepositoryPGX) DefinirOrigem(ctx context.Context, ordem int64, cliente, projeto *int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE production_orders SET customer_code = $3, project_id = $4, updated_at = NOW()
		WHERE id = $1 AND enterprise_id = $2`, ordem, empresa, cliente, projeto)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23503" {
			return domainerr.NewValidation("cliente ou projeto não encontrado nesta empresa")
		}
		return fmt.Errorf("gravando cliente e projeto da ordem: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domainerr.NewNotFound("ordem de produção não encontrada")
	}
	return nil
}

// ClienteDaEmpresa: o cliente existe na empresa da sessão (e o nome dele).
func (r *ProductionOrderRepositoryPGX) ClienteDaEmpresa(ctx context.Context, cliente int64) (string, bool, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return "", false, err
	}
	var nome string
	err = r.pool.QueryRow(ctx, `SELECT name FROM customers WHERE enterprise_id = $1 AND code = $2`, empresa, cliente).Scan(&nome)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("conferindo o cliente: %w", err)
	}
	return nome, true, nil
}
