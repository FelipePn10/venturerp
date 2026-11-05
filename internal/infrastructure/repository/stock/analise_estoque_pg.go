package stock

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// IdadeDosLotes lista o saldo em lote do mais antigo para o mais novo. A idade
// conta do recebimento; lote sem data de recebimento vem por último.
func (r *StockRepositorySQLC) IdadeDosLotes(ctx context.Context, almox *int64) ([]entity.IdadeDoLote, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT b.item_code, COALESCE(NULLIF(i.business_code,''), i.code::text), COALESCE(i.name,''), b.lot, b.warehouse_id, SUM(b.quantity),
		        ROUND(SUM(b.quantity) * COALESCE((SELECT MAX(s.avg_cost) FROM stock_balances s
		              WHERE s.enterprise_id=b.enterprise_id AND s.item_code=b.item_code AND s.mask=b.mask AND s.warehouse_id=b.warehouse_id), 0), 2),
		        MIN(l.received_at)
		   FROM stock_lot_balances b
		   JOIN items i ON i.code=b.item_code AND i.enterprise_id=b.enterprise_id
		   LEFT JOIN stock_lots l ON l.enterprise_id=b.enterprise_id AND l.item_code=b.item_code AND l.mask=b.mask AND l.lot=b.lot
		  WHERE b.enterprise_id=$1 AND b.quantity > 0 AND ($2::bigint IS NULL OR b.warehouse_id=$2)
		  GROUP BY b.enterprise_id, b.item_code, b.mask, i.business_code, i.code, i.name, b.lot, b.warehouse_id
		  ORDER BY MIN(l.received_at) NULLS LAST, 2, 4 LIMIT 1000`, empresa, almox)
	if err != nil {
		return nil, fmt.Errorf("listando a idade dos lotes: %w", err)
	}
	defer rows.Close()
	agora := time.Now()
	out := []entity.IdadeDoLote{}
	for rows.Next() {
		var l entity.IdadeDoLote
		if err := rows.Scan(&l.ItemCode, &l.Codigo, &l.Nome, &l.Lote, &l.WarehouseID, &l.Quantidade, &l.Valor, &l.Recebido); err != nil {
			return nil, fmt.Errorf("lendo a idade do lote: %w", err)
		}
		if l.Recebido != nil {
			d := int(agora.Sub(*l.Recebido).Hours() / 24)
			l.Dias = &d
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// NiveisPeloConsumo calcula o mínimo, a segurança e o máximo pelo consumo real da
// janela. Sem prazo informado, usa o do fornecedor preferencial do item.
func (r *StockRepositorySQLC) NiveisPeloConsumo(ctx context.Context, item int64, almox *int64, meses, prazo, diasSeguranca int) (*entity.NiveisSugeridos, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	desde := time.Now().AddDate(0, -meses, 0)
	var consumo decimal.Decimal
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(ABS(m.quantity)), 0) FROM stock_movements m
		  WHERE m.enterprise_id=$1 AND m.item_code=$2 AND ($3::bigint IS NULL OR m.warehouse_id=$3)
		    AND m.movement_date >= $4::date AND `+consumoSQL, empresa, item, almox, desde).Scan(&consumo); err != nil {
		return nil, fmt.Errorf("lendo o consumo do item: %w", err)
	}
	if prazo <= 0 {
		err := r.pool.QueryRow(ctx,
			`SELECT COALESCE(lead_time_days, 0) FROM item_preferred_suppliers
			  WHERE enterprise_id=$1 AND item_code=$2 AND is_active ORDER BY is_preferred DESC, ranking, id LIMIT 1`, empresa, item).Scan(&prazo)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("lendo o prazo do fornecedor: %w", err)
		}
	}
	n := entity.SugerirNiveis(consumo, int(time.Since(desde).Hours()/24), prazo, diasSeguranca)
	return &n, nil
}
