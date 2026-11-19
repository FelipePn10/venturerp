package aps

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// OrdensFilhas: as OFs filha abertas de cada OF pai, pela cadeia gravada ao
// firmar (production_chain_links). A pai é ligada pela OF ou, antes de a OF
// existir no vínculo, pela ordem planejada dela.
func (r *APSRepositorySQLC) OrdensFilhas(ctx context.Context, orderIDs []int64) (map[int64][]int64, error) {
	filhas := map[int64][]int64{}
	if r.pool == nil || len(orderIDs) == 0 {
		return filhas, nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT pai.id, filha.id
		  FROM production_orders pai
		  JOIN production_chain_links l ON l.enterprise_id = pai.enterprise_id
		       AND (l.parent_production_order_id = pai.id
		            OR (l.parent_production_order_id IS NULL AND l.parent_planned_order_id = pai.planned_order_id))
		  JOIN production_orders filha ON filha.enterprise_id = l.enterprise_id
		       AND filha.planned_order_id = l.child_planned_order_id AND filha.id <> pai.id AND filha.is_active
		       AND filha.status NOT IN ('COMPLETED', 'CLOSED', 'CANCELLED')
		 WHERE pai.enterprise_id = $1 AND pai.id = ANY($2)
		 ORDER BY pai.id, filha.id`, enterpriseID, orderIDs)
	if err != nil {
		return nil, fmt.Errorf("lendo a cadeia das OFs para o sequenciamento: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var pai, filha int64
		if err := rows.Scan(&pai, &filha); err != nil {
			return nil, err
		}
		filhas[pai] = append(filhas[pai], filha)
	}
	return filhas, rows.Err()
}

// FimProgramado: o término da última etapa já programada de cada OF.
func (r *APSRepositorySQLC) FimProgramado(ctx context.Context, orderIDs []int64) (map[int64]time.Time, error) {
	fins := map[int64]time.Time{}
	if r.pool == nil || len(orderIDs) == 0 {
		return fins, nil
	}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT production_order_id, MAX(scheduled_end) FROM production_sequences
		WHERE enterprise_id = $1 AND production_order_id = ANY($2) GROUP BY production_order_id`, enterpriseID, orderIDs)
	if err != nil {
		return nil, fmt.Errorf("lendo o término programado das OFs filha: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var fim time.Time
		if err := rows.Scan(&id, &fim); err != nil {
			return nil, err
		}
		fins[id] = fim
	}
	return fins, rows.Err()
}
