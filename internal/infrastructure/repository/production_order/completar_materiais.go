package production_order

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
)

// OFsSemMateriais lista, em TODAS as empresas, as OFs abertas ou em produção
// sem nenhum material de demanda cujo item tem estrutura ativa. É a fila da
// rotina que completa a lista; cada OF é tratada depois na sessão da empresa
// dela.
func (r *ProductionOrderRepositoryPGX) OFsSemMateriais(ctx context.Context, max int) ([]entity.OFSemMateriais, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.enterprise_id, e.code, o.item_code, COALESCE(o.mask, ''), o.planned_qty, o.start_date, o.created_by,
		       (COALESCE(o.produced_qty, 0) > 0 OR COALESCE(o.scrapped_qty, 0) > 0
		        OR EXISTS (SELECT 1 FROM production_appointments a WHERE a.production_order_id = o.id)
		        OR EXISTS (SELECT 1 FROM stock_movements sm WHERE sm.enterprise_id = o.enterprise_id
		                   AND sm.reference_type = 'PRODUCTION_ORDER' AND sm.reference_code = o.id)) AS ja_produziu
		FROM production_orders o
		JOIN enterprise e ON e.id = o.enterprise_id
		WHERE o.status IN ('OPEN', 'IN_PROGRESS') AND o.is_active IS NOT FALSE
		  AND NOT EXISTS (SELECT 1 FROM production_order_materials m
		                  WHERE m.production_order_id = o.id AND m.enterprise_id = o.enterprise_id
		                    AND m.material_kind = 'DEMAND')
		  AND EXISTS (SELECT 1 FROM item_structures s JOIN items i ON i.code = s.parent_code
		              WHERE s.parent_code = o.item_code AND s.is_active AND i.enterprise_id = o.enterprise_id)
		ORDER BY o.id
		LIMIT $1`, max)
	if err != nil {
		return nil, fmt.Errorf("listando as OFs sem lista de materiais: %w", err)
	}
	defer rows.Close()
	var out []entity.OFSemMateriais
	for rows.Next() {
		var of entity.OFSemMateriais
		var inicio *time.Time
		if err := rows.Scan(&of.ID, &of.EnterpriseID, &of.EnterpriseCode, &of.ItemCode, &of.Mask, &of.PlannedQty, &inicio, &of.CreatedBy, &of.JaProduziu); err != nil {
			return nil, err
		}
		of.StartDate = inicio
		out = append(out, of)
	}
	return out, rows.Err()
}

// CompletarMateriais grava a lista de materiais da OF que está sem ela, com a
// etapa do roteiro de cada material. Confere a empresa da sessão e, com a OF
// travada, só grava se ela continuar aberta/em produção e sem lista — rodar de
// novo, ou em paralelo, não duplica. Devolve quantos materiais entraram.
func (r *ProductionOrderRepositoryPGX) CompletarMateriais(ctx context.Context, ordemID int64, materiais []*entity.ProductionOrderMaterial) (int, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var itemCode int64
	var mask string
	err = tx.QueryRow(ctx, `SELECT item_code, COALESCE(mask, '') FROM production_orders
		WHERE id = $1 AND enterprise_id = $2 AND status IN ('OPEN', 'IN_PROGRESS')
		  AND NOT EXISTS (SELECT 1 FROM production_order_materials m
		                  WHERE m.production_order_id = $1 AND m.enterprise_id = $2 AND m.material_kind = 'DEMAND')
		FOR UPDATE`, ordemID, enterpriseID).Scan(&itemCode, &mask)
	if err == pgx.ErrNoRows {
		return 0, nil // outra empresa, encerrada ou já completada
	}
	if err != nil {
		return 0, err
	}
	n := 0
	for _, m := range materiais {
		if !m.Quantity.IsPositive() {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO production_order_materials
			(production_order_id,enterprise_id,material_kind,item_code,mask,quantity,warehouse_id,automatic_issue,notes,created_by)
			VALUES ($1,$2,'DEMAND',$3,$4,$5,$6,$7,$8,$9)`, ordemID, enterpriseID, m.ItemCode, m.Mask, m.Quantity,
			m.WarehouseID, m.AutomaticIssue, m.Notes, m.CreatedBy); err != nil {
			return 0, fmt.Errorf("gravando o material %d da OF %d: %w", m.ItemCode, ordemID, err)
		}
		n++
	}
	if err := gravarEtapaDosMateriaisTx(ctx, tx, enterpriseID, ordemID, itemCode, mask); err != nil {
		return 0, err
	}
	return n, tx.Commit(ctx)
}
