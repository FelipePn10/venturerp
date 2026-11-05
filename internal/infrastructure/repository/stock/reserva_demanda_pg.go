package stock

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

// Reserva do produzido para a demanda independente de finalidade própria
// (amostra, feira, consumo interno, assistência, projeto). A peça que a OF da
// demanda entrega fica reservada para ela — nenhum pedido de venda nem onda de
// separação a pega — até a saída para a finalidade ("Atender"), que consome a
// reserva, ou até a demanda ser encerrada, que a libera. Estornada a saída, a
// peça volta reservada. Ver .ai/tasks/paridade-mrp-lote-reserva-excecao.md.
//
// A reserva fica na linha exata em que a peça entrou — almoxarifado, endereço e
// lote —, porque é a linha de lote que a separação por onda consulta.

const motivosDeFinalidade = `('AMOSTRA', 'FEIRA', 'CONSUMO_INTERNO', 'ASSISTENCIA', 'PROJETO')`

// faltaDaDemanda: quantidade − saídas não estornadas − reservas ativas.
const faltaDaDemanda = `d.quantity
	- COALESCE((SELECT SUM(ABS(m.quantity)) FROM stock_movements m
	             WHERE m.enterprise_id = d.enterprise_id AND m.reference_type = 'DEMANDA_INDEPENDENTE' AND m.reference_code = d.code
	               AND m.movement_type = 'OUT' AND m.reversed_by_movement_id IS NULL), 0)
	- COALESCE((SELECT SUM(r.quantity) FROM stock_reservations r
	             WHERE r.enterprise_id = d.enterprise_id AND r.reference_type = 'DEMANDA_INDEPENDENTE' AND r.reference_code = d.code
	               AND r.status = 'ACTIVE'), 0)`

type faltaDeDemanda struct {
	code int64
	qtd  decimal.Decimal
}

// ReservarEntradaDaOFTx reserva o que o movimento de entrada da OF trouxe para
// as demandas de finalidade ligadas a ela (direto ou pela ordem planejada), na
// ordem da data de cada demanda, até o que falta a cada uma. Roda na transação
// da entrega, depois do movimento. Devolve o total reservado.
func ReservarEntradaDaOFTx(ctx context.Context, tx pgx.Tx, enterpriseID, productionOrderID int64, m *entity.StockMovement, por uuid.UUID) (decimal.Decimal, error) {
	entrou := entity.SignedQuantity(m.MovementType, m.Quantity)
	if !entrou.IsPositive() {
		return decimal.Zero, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT d.code, `+faltaDaDemanda+`
		  FROM independent_demand_orders v
		  JOIN independent_demands d ON d.id = v.demand_id AND d.enterprise_id = v.enterprise_id
		 WHERE v.enterprise_id = $1 AND d.situacao = 'ABERTA' AND d.motivo IN `+motivosDeFinalidade+`
		   AND d.item_code = $3
		   AND (v.production_order_id = $2
		        OR v.planned_order_id = (SELECT planned_order_id FROM production_orders WHERE id = $2 AND enterprise_id = $1))
		 GROUP BY d.code, d.quantity, d.enterprise_id, d.demand_date
		 ORDER BY d.demand_date, d.code`, enterpriseID, productionOrderID, m.ItemCode)
	if err != nil {
		return decimal.Zero, fmt.Errorf("lendo as demandas da OF %d: %w", productionOrderID, err)
	}
	faltas, err := lerFaltas(rows)
	if err != nil {
		return decimal.Zero, err
	}
	reservado := decimal.Zero
	for _, f := range faltas {
		resta := entrou.Sub(reservado)
		if !resta.IsPositive() {
			break
		}
		nota := fmt.Sprintf("Produzido pela OF %d para a demanda %d", productionOrderID, f.code)
		feito, err := reservarParaDemandaTx(ctx, tx, enterpriseID, f.code, m, decimal.Min(f.qtd, resta), nota, por)
		if err != nil {
			return reservado, err
		}
		if !feito.IsPositive() {
			break // nada livre na linha
		}
		reservado = reservado.Add(feito)
	}
	return reservado, nil
}

// ReservarDeVoltaDoEstornoTx: estornada a saída para a demanda, a peça que
// voltou ao estoque fica de novo reservada para ela, até o que lhe falta.
func ReservarDeVoltaDoEstornoTx(ctx context.Context, tx pgx.Tx, enterpriseID, demandCode int64, m *entity.StockMovement, por uuid.UUID) (decimal.Decimal, error) {
	entrou := entity.SignedQuantity(m.MovementType, m.Quantity)
	if !entrou.IsPositive() {
		return decimal.Zero, nil
	}
	rows, err := tx.Query(ctx, `SELECT d.code, `+faltaDaDemanda+`
		  FROM independent_demands d
		 WHERE d.enterprise_id = $1 AND d.code = $2 AND d.item_code = $3
		   AND d.situacao = 'ABERTA' AND d.motivo IN `+motivosDeFinalidade, enterpriseID, demandCode, m.ItemCode)
	if err != nil {
		return decimal.Zero, fmt.Errorf("lendo a demanda %d: %w", demandCode, err)
	}
	faltas, err := lerFaltas(rows)
	if err != nil || len(faltas) == 0 {
		return decimal.Zero, err
	}
	nota := fmt.Sprintf("Estorno da saída para a demanda %d", demandCode)
	return reservarParaDemandaTx(ctx, tx, enterpriseID, demandCode, m, decimal.Min(faltas[0].qtd, entrou), nota, por)
}

func lerFaltas(rows pgx.Rows) ([]faltaDeDemanda, error) {
	defer rows.Close()
	var faltas []faltaDeDemanda
	for rows.Next() {
		var f faltaDeDemanda
		if err := rows.Scan(&f.code, &f.qtd); err != nil {
			return nil, fmt.Errorf("lendo o saldo da demanda: %w", err)
		}
		if f.qtd.IsPositive() {
			faltas = append(faltas, f)
		}
	}
	return faltas, rows.Err()
}

func loteDoMovimento(m *entity.StockMovement) string {
	if m.Lot == nil {
		return ""
	}
	return strings.TrimSpace(*m.Lot)
}

// reservarParaDemandaTx reserva até qtd na linha de saldo do movimento
// (almoxarifado + endereço; e a do lote, quando há), limitada ao que está livre
// nela, e grava a reserva com endereço e lote. Devolve o reservado.
func reservarParaDemandaTx(ctx context.Context, tx pgx.Tx, enterpriseID, demandCode int64, m *entity.StockMovement,
	qtd decimal.Decimal, nota string, por uuid.UUID) (decimal.Decimal, error) {
	if !qtd.IsPositive() {
		return decimal.Zero, nil
	}
	endereco, lote := enderecoDoSaldo(m), loteDoMovimento(m)
	var saldoID int64
	var livre decimal.Decimal
	err := tx.QueryRow(ctx, `SELECT id, quantity - reserved_qty FROM stock_balances
		WHERE enterprise_id = $1 AND item_code = $2 AND mask = $3 AND warehouse_id = $4 AND address = $5
		FOR UPDATE`, enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, endereco).Scan(&saldoID, &livre)
	if err == pgx.ErrNoRows {
		return decimal.Zero, nil
	}
	if err != nil {
		return decimal.Zero, fmt.Errorf("lendo o saldo livre: %w", err)
	}
	q := decimal.Min(qtd, livre)
	var loteID int64
	if lote != "" {
		var livreLote decimal.Decimal
		err := tx.QueryRow(ctx, `SELECT id, quantity - reserved_qty FROM stock_lot_balances
			WHERE enterprise_id = $1 AND item_code = $2 AND mask = $3 AND warehouse_id = $4 AND lot = $5 AND address = $6
			FOR UPDATE`, enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, lote, endereco).Scan(&loteID, &livreLote)
		if err == pgx.ErrNoRows {
			return decimal.Zero, nil
		}
		if err != nil {
			return decimal.Zero, fmt.Errorf("lendo o saldo livre do lote: %w", err)
		}
		q = decimal.Min(q, livreLote)
	}
	if !q.IsPositive() {
		return decimal.Zero, nil
	}
	if _, err := tx.Exec(ctx, `UPDATE stock_balances SET reserved_qty = reserved_qty + $2, updated_at = NOW() WHERE id = $1`, saldoID, q); err != nil {
		return decimal.Zero, fmt.Errorf("reservando no saldo: %w", err)
	}
	if loteID != 0 {
		if _, err := tx.Exec(ctx, `UPDATE stock_lot_balances SET reserved_qty = reserved_qty + $2, updated_at = NOW() WHERE id = $1`, loteID, q); err != nil {
			return decimal.Zero, fmt.Errorf("reservando no saldo do lote: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO stock_reservations
		(item_code, mask, warehouse_id, address, lot, quantity, reference_type, reference_code, reservation_date, status, notes, created_by, enterprise_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_DATE, 'ACTIVE', $9, $10, $11)`,
		m.ItemCode, m.Mask, m.WarehouseID, endereco, lote, q, entity.ReferenceTypeIndependentDemand, demandCode, nota, por, enterpriseID); err != nil {
		return decimal.Zero, fmt.Errorf("reservando para a demanda %d: %w", demandCode, err)
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
	var item int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MIN(item_code), 0) FROM stock_reservations
		 WHERE enterprise_id = $1 AND reference_type = $2 AND reference_code = $3 AND status = 'ACTIVE'`,
		enterpriseID, entity.ReferenceTypeIndependentDemand, demandCode).Scan(&item); err != nil {
		return liberado, fmt.Errorf("lendo o item das reservas da demanda %d: %w", demandCode, err)
	}
	if item != 0 {
		if err := TravarItensTx(ctx, tx, enterpriseID, item); err != nil {
			return liberado, err
		}
	}
	rows, err := tx.Query(ctx, `SELECT id, item_code, mask, warehouse_id, COALESCE(address, ''), COALESCE(lot, ''), quantity
		  FROM stock_reservations
		 WHERE enterprise_id = $1 AND reference_type = $2 AND reference_code = $3 AND status = 'ACTIVE'
		   AND ($4::bigint IS NULL OR warehouse_id = $4)
		 ORDER BY id FOR UPDATE`, enterpriseID, entity.ReferenceTypeIndependentDemand, demandCode, warehouseID)
	if err != nil {
		return liberado, fmt.Errorf("lendo as reservas da demanda %d: %w", demandCode, err)
	}
	type reserva struct {
		id, item, almox      int64
		mask, endereco, lote string
		qtd                  decimal.Decimal
	}
	var reservas []reserva
	for rows.Next() {
		var r reserva
		if err := rows.Scan(&r.id, &r.item, &r.mask, &r.almox, &r.endereco, &r.lote, &r.qtd); err != nil {
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
		if _, err := tx.Exec(ctx, `UPDATE stock_balances SET reserved_qty = GREATEST(reserved_qty - $6, 0), updated_at = NOW()
			WHERE enterprise_id = $1 AND item_code = $2 AND mask = $3 AND warehouse_id = $4 AND address = $5`,
			enterpriseID, r.item, r.mask, r.almox, r.endereco, q); err != nil {
			return liberado, fmt.Errorf("liberando o saldo reservado: %w", err)
		}
		if r.lote != "" {
			if _, err := tx.Exec(ctx, `UPDATE stock_lot_balances SET reserved_qty = GREATEST(reserved_qty - $7, 0), updated_at = NOW()
				WHERE enterprise_id = $1 AND item_code = $2 AND mask = $3 AND warehouse_id = $4 AND lot = $5 AND address = $6`,
				enterpriseID, r.item, r.mask, r.almox, r.lote, r.endereco, q); err != nil {
				return liberado, fmt.Errorf("liberando o saldo reservado do lote: %w", err)
			}
		}
		liberado = liberado.Add(q)
	}
	return liberado, nil
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
