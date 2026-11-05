package stock

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// ContagensParaAcuracidade lê as contagens encerradas nos últimos meses: linha de
// inventário fechado, com a última contagem, e contagem cíclica concluída. O mês
// é o da empresa.
func (r *StockRepositorySQLC) ContagensParaAcuracidade(ctx context.Context, meses int, almox *int64) ([]entity.ContagemAcuracidade, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	fuso, err := r.fusoDaEmpresa(ctx, empresa)
	if err != nil {
		return nil, err
	}
	desde := time.Now().In(fuso).AddDate(0, -meses, 0)
	rows, err := r.pool.Query(ctx,
		`WITH contagens AS (
		   SELECT COALESCE(p.end_date::timestamptz, p.updated_at) quando, l.item_code, l.warehouse_id, l.address,
		          l.system_qty esperado, l.counted_qty contado, COALESCE(l.unit_cost, 0) custo
		     FROM physical_inventory_items l JOIN physical_inventories p ON p.id=l.inventory_id
		    WHERE p.enterprise_id=$1 AND p.status=$4 AND l.counted_qty IS NOT NULL
		   UNION ALL
		   SELECT COALESCE(c.completed_at, c.updated_at), c.item_code, c.warehouse_id, c.address,
		          c.expected_quantity, c.counted_quantity,
		          COALESCE((SELECT MAX(b.avg_cost) FROM stock_balances b
		                     WHERE b.enterprise_id=c.enterprise_id AND b.item_code=c.item_code AND b.warehouse_id=c.warehouse_id), 0)
		     FROM stock_cycle_counts c
		    WHERE c.enterprise_id=$1 AND c.state IN ('DIVERGENTE','CONCLUIDA','APROVADA')
		      AND c.counted_quantity IS NOT NULL AND c.expected_quantity IS NOT NULL)
		 SELECT x.quando, x.item_code, COALESCE(NULLIF(i.business_code,''), i.code::text), COALESCE(i.name,''),
		        COALESCE(i.planning_abc_class::text,''), x.warehouse_id, x.address, x.esperado, x.contado, x.custo
		   FROM contagens x JOIN items i ON i.enterprise_id=$1 AND i.code=x.item_code
		  WHERE x.quando >= $2 AND ($3::bigint IS NULL OR x.warehouse_id=$3)`,
		empresa, desde, almox, entity.InventarioFechado)
	if err != nil {
		return nil, fmt.Errorf("reading counts for accuracy: %w", err)
	}
	defer rows.Close()
	out := []entity.ContagemAcuracidade{}
	for rows.Next() {
		var c entity.ContagemAcuracidade
		var quando time.Time
		if err := rows.Scan(&quando, &c.ItemCode, &c.Codigo, &c.Nome, &c.Classe, &c.WarehouseID, &c.Endereco, &c.Esperado, &c.Contado, &c.Custo); err != nil {
			return nil, fmt.Errorf("reading counts for accuracy: %w", err)
		}
		c.Mes = quando.In(fuso).Format("2006-01")
		out = append(out, c)
	}
	return out, rows.Err()
}
