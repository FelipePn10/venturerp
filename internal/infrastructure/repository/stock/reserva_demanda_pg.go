package stock

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

// Reserva do produzido para a demanda independente de finalidade própria
// (amostra, feira, consumo interno, assistência, projeto). A peça que a OF da
// demanda entrega fica reservada para ela — nenhum pedido de venda a pega — até
// a saída para a finalidade ("Atender"), que consome a reserva, ou até a
// demanda ser encerrada, que a libera. Ver
// .ai/tasks/paridade-mrp-lote-reserva-excecao.md.

const motivosDeFinalidade = `('AMOSTRA', 'FEIRA', 'CONSUMO_INTERNO', 'ASSISTENCIA', 'PROJETO')`

// ReservarProduzidoParaDemandasTx reserva a entrada da OF para as demandas de
// finalidade ligadas a ela (direto ou pela ordem planejada), na ordem da data
// de cada demanda, até o que falta a cada uma (quantidade − saída − já
// reservado) e até o que está livre no almoxarifado. Roda na transação da
// entrega, depois do movimento de entrada. Devolve o total reservado.
func ReservarProduzidoParaDemandasTx(ctx context.Context, tx pgx.Tx, enterpriseID, productionOrderID, itemCode int64, mask string,
	warehouseID int64, qtd decimal.Decimal, por uuid.UUID) (decimal.Decimal, error) {
	reservado := decimal.Zero
	if !qtd.IsPositive() {
		return reservado, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT d.code, d.quantity
		       - COALESCE((SELECT SUM(ABS(m.quantity)) FROM stock_movements m
		                    WHERE m.enterprise_id = d.enterprise_id AND m.reference_type = $4 AND m.reference_code = d.code
		                      AND m.movement_type = 'OUT' AND m.reversed_by_movement_id IS NULL), 0)
		       - COALESCE((SELECT SUM(r.quantity) FROM stock_reservations r
		                    WHERE r.enterprise_id = d.enterprise_id AND r.reference_type = $4 AND r.reference_code = d.code
		                      AND r.status = 'ACTIVE'), 0)
		  FROM independent_demand_orders v
		  JOIN independent_demands d ON d.id = v.demand_id AND d.enterprise_id = v.enterprise_id
		 WHERE v.enterprise_id = $1 AND d.situacao = 'ABERTA' AND d.motivo IN `+motivosDeFinalidade+`
		   AND d.item_code = $3
		   AND (v.production_order_id = $2
		        OR v.planned_order_id = (SELECT planned_order_id FROM production_orders WHERE id = $2 AND enterprise_id = $1))
		 GROUP BY d.code, d.quantity, d.enterprise_id, d.demand_date
		 ORDER BY d.demand_date, d.code`, enterpriseID, productionOrderID, itemCode, entity.ReferenceTypeIndependentDemand)
	if err != nil {
		return reservado, fmt.Errorf("lendo as demandas da OF %d: %w", productionOrderID, err)
	}
	type falta struct {
		code int64
		qtd  decimal.Decimal
	}
	var faltas []falta
	for rows.Next() {
		var f falta
		if err := rows.Scan(&f.code, &f.qtd); err != nil {
			rows.Close()
			return reservado, fmt.Errorf("lendo a demanda da OF: %w", err)
		}
		if f.qtd.IsPositive() {
			faltas = append(faltas, f)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return reservado, err
	}
	resta := qtd
	for _, f := range faltas {
		if !resta.IsPositive() {
			break
		}
		q := decimal.Min(f.qtd, resta)
		feito, err := reservarNoSaldoTx(ctx, tx, enterpriseID, itemCode, mask, warehouseID, q)
		if err != nil {
			return reservado, err
		}
		if !feito.IsPositive() {
			break // nada livre no almoxarifado
		}
		code := f.code
		ref := entity.ReferenceTypeIndependentDemand
		nota := fmt.Sprintf("Produzido pela OF %d para a demanda %d", productionOrderID, code)
		if _, err := tx.Exec(ctx, `INSERT INTO stock_reservations
			(item_code, mask, warehouse_id, quantity, reference_type, reference_code, reservation_date, status, notes, created_by, enterprise_id)
			VALUES ($1, $2, $3, $4, $5, $6, CURRENT_DATE, 'ACTIVE', $7, $8, $9)`,
			itemCode, mask, warehouseID, feito, ref, code, nota, por, enterpriseID); err != nil {
			return reservado, fmt.Errorf("reservando o produzido para a demanda %d: %w", code, err)
		}
		reservado = reservado.Add(feito)
		resta = resta.Sub(feito)
	}
	return reservado, nil
}

// reservarNoSaldoTx soma a reserva numa linha de saldo do almoxarifado (a de
// mais saldo livre), até o que estiver livre nela. Devolve o reservado.
func reservarNoSaldoTx(ctx context.Context, tx pgx.Tx, enterpriseID, itemCode int64, mask string, warehouseID int64, qtd decimal.Decimal) (decimal.Decimal, error) {
	var id int64
	var livre decimal.Decimal
	err := tx.QueryRow(ctx, `SELECT id, quantity - reserved_qty FROM stock_balances
		WHERE enterprise_id = $1 AND item_code = $2 AND mask = $3 AND warehouse_id = $4 AND quantity - reserved_qty > 0
		ORDER BY quantity - reserved_qty DESC, id LIMIT 1 FOR UPDATE`, enterpriseID, itemCode, mask, warehouseID).Scan(&id, &livre)
	if err == pgx.ErrNoRows {
		return decimal.Zero, nil
	}
	if err != nil {
		return decimal.Zero, fmt.Errorf("lendo o saldo livre: %w", err)
	}
	q := decimal.Min(qtd, livre)
	if _, err := tx.Exec(ctx, `UPDATE stock_balances SET reserved_qty = reserved_qty + $2, updated_at = NOW() WHERE id = $1`, id, q); err != nil {
		return decimal.Zero, fmt.Errorf("reservando no saldo: %w", err)
	}
	return q, nil
}

// LiberarReservasDaDemandaTx tira as reservas ativas da demanda: até qtd
// (nil = todas) no almoxarifado (nil = qualquer). consumir marca CONSUMED (a
// saída para a finalidade usou a peça); senão CANCELLED (a demanda encerrou).
// Devolve o liberado.
func LiberarReservasDaDemandaTx(ctx context.Context, tx pgx.Tx, enterpriseID, demandCode int64, warehouseID *int64,
	qtd *decimal.Decimal, consumir bool) (decimal.Decimal, error) {
	liberado := decimal.Zero
	rows, err := tx.Query(ctx, `SELECT id, item_code, mask, warehouse_id, quantity FROM stock_reservations
		WHERE enterprise_id = $1 AND reference_type = $2 AND reference_code = $3 AND status = 'ACTIVE'
		  AND ($4::bigint IS NULL OR warehouse_id = $4)
		ORDER BY id FOR UPDATE`, enterpriseID, entity.ReferenceTypeIndependentDemand, demandCode, warehouseID)
	if err != nil {
		return liberado, fmt.Errorf("lendo as reservas da demanda %d: %w", demandCode, err)
	}
	type reserva struct {
		id, item, almox int64
		mask            string
		qtd             decimal.Decimal
	}
	var reservas []reserva
	for rows.Next() {
		var r reserva
		if err := rows.Scan(&r.id, &r.item, &r.mask, &r.almox, &r.qtd); err != nil {
			rows.Close()
			return liberado, fmt.Errorf("lendo a reserva da demanda: %w", err)
		}
		reservas = append(reservas, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return liberado, err
	}
	estado := "CANCELLED"
	if consumir {
		estado = "CONSUMED"
	}
	for _, r := range reservas {
		q := r.qtd
		if qtd != nil {
			q = decimal.Min(q, qtd.Sub(liberado))
		}
		if !q.IsPositive() {
			break
		}
		if q.Equal(r.qtd) {
			if _, err := tx.Exec(ctx, `UPDATE stock_reservations SET status = $2, updated_at = NOW() WHERE id = $1`, r.id, estado); err != nil {
				return liberado, fmt.Errorf("fechando a reserva %d: %w", r.id, err)
			}
		} else if _, err := tx.Exec(ctx, `UPDATE stock_reservations SET quantity = quantity - $2, updated_at = NOW() WHERE id = $1`, r.id, q); err != nil {
			return liberado, fmt.Errorf("reduzindo a reserva %d: %w", r.id, err)
		}
		if err := tirarDoSaldoReservadoTx(ctx, tx, enterpriseID, r.item, r.mask, r.almox, q); err != nil {
			return liberado, err
		}
		liberado = liberado.Add(q)
	}
	return liberado, nil
}

// tirarDoSaldoReservadoTx devolve qtd do reservado das linhas de saldo do
// almoxarifado que têm reserva.
func tirarDoSaldoReservadoTx(ctx context.Context, tx pgx.Tx, enterpriseID, itemCode int64, mask string, warehouseID int64, qtd decimal.Decimal) error {
	rows, err := tx.Query(ctx, `SELECT id, reserved_qty FROM stock_balances
		WHERE enterprise_id = $1 AND item_code = $2 AND mask = $3 AND warehouse_id = $4 AND reserved_qty > 0
		ORDER BY reserved_qty DESC, id FOR UPDATE`, enterpriseID, itemCode, mask, warehouseID)
	if err != nil {
		return fmt.Errorf("lendo o saldo reservado: %w", err)
	}
	type linha struct {
		id  int64
		res decimal.Decimal
	}
	var linhas []linha
	for rows.Next() {
		var l linha
		if err := rows.Scan(&l.id, &l.res); err != nil {
			rows.Close()
			return fmt.Errorf("lendo o saldo reservado: %w", err)
		}
		linhas = append(linhas, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	resta := qtd
	for _, l := range linhas {
		if !resta.IsPositive() {
			break
		}
		q := decimal.Min(resta, l.res)
		if _, err := tx.Exec(ctx, `UPDATE stock_balances SET reserved_qty = reserved_qty - $2, updated_at = NOW() WHERE id = $1`, l.id, q); err != nil {
			return fmt.Errorf("liberando o saldo reservado: %w", err)
		}
		resta = resta.Sub(q)
	}
	return nil
}

// consumirReservaDaSaidaTx: a saída para a finalidade da demanda usa a peça
// reservada para ela. Roda antes de o movimento baixar o saldo.
func consumirReservaDaSaidaTx(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	if m.MovementType != entity.MovementTypeOut || m.ReferenceType == nil || *m.ReferenceType != entity.ReferenceTypeIndependentDemand ||
		m.ReferenceCode == nil {
		return nil
	}
	almox := m.WarehouseID
	q := m.Quantity.Abs()
	_, err := LiberarReservasDaDemandaTx(ctx, tx, enterpriseID, *m.ReferenceCode, &almox, &q, true)
	return err
}
