package shipment

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// BaixaPelaNFe lê o que do romaneio a NF-e de saída já tirou do estoque. Só
// conta nota autorizada que gerou a saída de estoque (NF_SAIDA): tipo de nota
// que não movimenta estoque deixa a mercadoria na prateleira, e a separação
// tem de reservá-la.
func (r *ShipmentRepositoryPG) BaixaPelaNFe(ctx context.Context, shipmentCode int64) (repository.BaixaPelaNFe, error) {
	out := repository.BaixaPelaNFe{PorLinhaDoPedido: map[int64]decimal.Decimal{}}
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return out, err
	}
	if err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM shipments s
			JOIN fiscal_exits fe ON fe.id = s.fiscal_exit_id AND fe.enterprise_id = s.enterprise_id
			WHERE s.code = $1 AND s.enterprise_id = $2 AND fe.status = 'AUTHORIZED'
			  AND EXISTS (SELECT 1 FROM stock_movements m
			              WHERE m.enterprise_id = $2 AND m.reference_type = 'NF_SAIDA' AND m.reference_code = fe.id))`,
		shipmentCode, enterpriseID).Scan(&out.NotaVinculadaBaixou); err != nil {
		return out, fmt.Errorf("lendo a NF-e do romaneio %d: %w", shipmentCode, err)
	}
	rows, err := r.pool.Query(ctx, `
		WITH linhas AS (
			SELECT DISTINCT si.sales_order_item_code AS linha
			FROM shipment_items si JOIN shipments s ON s.id = si.shipment_id
			WHERE s.code = $1 AND s.enterprise_id = $2 AND si.sales_order_item_code IS NOT NULL
		), faturado AS (
			SELECT fi.sales_order_item_code AS linha, SUM(fi.quantity) AS qtd
			FROM fiscal_exit_items fi
			JOIN fiscal_exits fe ON fe.id = fi.fiscal_exit_id
			WHERE fe.enterprise_id = $2 AND fe.status = 'AUTHORIZED'
			  AND fi.sales_order_item_code IN (SELECT linha FROM linhas)
			  AND EXISTS (SELECT 1 FROM stock_movements m
			              WHERE m.enterprise_id = $2 AND m.reference_type = 'NF_SAIDA' AND m.reference_code = fe.id)
			GROUP BY fi.sales_order_item_code
		), coberto AS (
			SELECT si.sales_order_item_code AS linha, SUM(si.quantity) AS qtd
			FROM shipment_items si JOIN shipments s ON s.id = si.shipment_id
			WHERE s.enterprise_id = $2 AND s.code <> $1
			  AND s.status IN ('SEPARATED', 'CONFERRED', 'SHIPPED')
			  AND si.sales_order_item_code IN (SELECT linha FROM linhas)
			GROUP BY si.sales_order_item_code
		)
		SELECT f.linha, GREATEST(f.qtd - COALESCE(c.qtd, 0), 0)
		FROM faturado f LEFT JOIN coberto c ON c.linha = f.linha`,
		shipmentCode, enterpriseID)
	if err != nil {
		return out, fmt.Errorf("lendo o faturado das linhas do romaneio %d: %w", shipmentCode, err)
	}
	defer rows.Close()
	for rows.Next() {
		var linha int64
		var qtd decimal.Decimal
		if err := rows.Scan(&linha, &qtd); err != nil {
			return out, err
		}
		if qtd.IsPositive() {
			out.PorLinhaDoPedido[linha] = qtd
		}
	}
	return out, rows.Err()
}
