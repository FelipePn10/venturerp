package stock

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// soltarReservaDoPedidoTx tira até qtd da reserva do pedido no almoxarifado e
// devolve o quanto tirou. Pedido de outra empresa não passa daqui.
func soltarReservaDoPedidoTx(ctx context.Context, tx pgx.Tx, empresa, pedido, item int64, mask string, almox int64, qtd decimal.Decimal) (decimal.Decimal, error) {
	if err := TravarItensTx(ctx, tx, empresa, item); err != nil {
		return decimal.Zero, err
	}
	var existe bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM sales_orders o JOIN enterprise e ON e.code=o.enterprise_code WHERE o.code=$1 AND e.id=$2 AND o.is_active)`,
		pedido, empresa).Scan(&existe); err != nil {
		return decimal.Zero, fmt.Errorf("checking sales order: %w", err)
	}
	if !existe {
		return decimal.Zero, errorsuc.NewValidationError(fmt.Sprintf("o pedido %d não existe nesta empresa", pedido))
	}
	rows, err := tx.Query(ctx,
		`SELECT id, quantity FROM stock_reservations
		  WHERE enterprise_id=$1 AND reference_type=$2 AND reference_code=$3 AND item_code=$4 AND mask=$5 AND warehouse_id=$6
		    AND status='ACTIVE' ORDER BY id FOR UPDATE`,
		empresa, entity.ReferenceTypeSalesOrder, pedido, item, mask, almox)
	if err != nil {
		return decimal.Zero, fmt.Errorf("reading order reservations: %w", err)
	}
	type reserva struct {
		id  int64
		qtd decimal.Decimal
	}
	var lista []reserva
	for rows.Next() {
		var v reserva
		if err := rows.Scan(&v.id, &v.qtd); err != nil {
			rows.Close()
			return decimal.Zero, fmt.Errorf("reading order reservations: %w", err)
		}
		lista = append(lista, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return decimal.Zero, fmt.Errorf("reading order reservations: %w", err)
	}
	solto := decimal.Zero
	for _, v := range lista {
		q := decimal.Min(v.qtd, qtd.Sub(solto))
		if !q.IsPositive() {
			break
		}
		if q.Equal(v.qtd) {
			_, err = tx.Exec(ctx, `UPDATE stock_reservations SET status='CANCELLED', updated_at=NOW() WHERE id=$1`, v.id)
		} else {
			_, err = tx.Exec(ctx, `UPDATE stock_reservations SET quantity = quantity - $2, updated_at=NOW() WHERE id=$1`, v.id, q)
		}
		if err != nil {
			return decimal.Zero, fmt.Errorf("releasing order reservation: %w", err)
		}
		solto = solto.Add(q)
	}
	if solto.IsPositive() {
		if err := adjustReservedTx(ctx, tx, empresa, item, mask, almox, solto.Neg()); err != nil {
			return decimal.Zero, err
		}
	}
	return solto, nil
}

// reservarParaPedidoTx devolve a reserva ao pedido, do jeito que a nota a consome.
// A onda solta a reserva do pedido por produto; na volta ela é repartida pelas
// LINHAS desse produto (na ordem do pedido, até o saldo em aberto de cada uma
// que ainda não está reservado), como a confirmação grava. Gravava o código do
// produto em reference_item_code: com o produto em mais de uma linha, cancelar
// ou reduzir uma linha não achava a reserva dela. O que passar do saldo das
// linhas segue com o código do produto, como antes.
func reservarParaPedidoTx(ctx context.Context, tx pgx.Tx, empresa, pedido, item int64, mask string, almox int64, qtd decimal.Decimal, ator uuid.UUID) error {
	partes, err := linhasParaReservaTx(ctx, tx, empresa, pedido, item, mask, qtd)
	if err != nil {
		return err
	}
	for _, p := range partes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO stock_reservations (item_code, mask, warehouse_id, quantity, reference_type, reference_code,
			     reference_item_code, status, created_by, enterprise_id)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,'ACTIVE',$8,$9)`,
			item, mask, almox, p.qtd, entity.ReferenceTypeSalesOrder, pedido, p.ref, ator, empresa); err != nil {
			return fmt.Errorf("restoring order reservation: %w", err)
		}
	}
	return adjustReservedTx(ctx, tx, empresa, item, mask, almox, qtd)
}

type parteDaReserva struct {
	ref int64
	qtd decimal.Decimal
}

// linhasParaReservaTx reparte qtd pelas linhas em aberto do produto no pedido.
func linhasParaReservaTx(ctx context.Context, tx pgx.Tx, empresa, pedido, item int64, mask string, qtd decimal.Decimal) ([]parteDaReserva, error) {
	rows, err := tx.Query(ctx,
		`SELECT i.code,
		        GREATEST(i.requested_qty - i.attended_qty - i.cancelled_qty
		                 - COALESCE((SELECT SUM(r.quantity) FROM stock_reservations r
		                              WHERE r.enterprise_id = $1 AND r.reference_type = $5 AND r.reference_code = i.sales_order_code
		                                AND r.reference_item_code = i.code AND r.item_code = i.item_code AND r.status = 'ACTIVE'), 0), 0)
		   FROM sales_order_items i
		   JOIN sales_orders o ON o.code = i.sales_order_code
		   JOIN enterprise e ON e.code = o.enterprise_code AND e.id = $1
		  WHERE i.sales_order_code = $2 AND i.item_code = $3 AND COALESCE(i.mask, '') = $4
		    AND i.is_active AND i.status IN ('OPEN', 'PARTIAL')
		  ORDER BY i.sequence, i.code`,
		empresa, pedido, item, mask, entity.ReferenceTypeSalesOrder)
	if err != nil {
		return nil, fmt.Errorf("reading order lines for the reservation: %w", err)
	}
	defer rows.Close()
	var partes []parteDaReserva
	resta := qtd
	for rows.Next() {
		var code int64
		var falta decimal.Decimal
		if err := rows.Scan(&code, &falta); err != nil {
			return nil, fmt.Errorf("reading order lines for the reservation: %w", err)
		}
		if q := decimal.Min(falta, resta); q.IsPositive() {
			partes = append(partes, parteDaReserva{ref: code, qtd: q})
			resta = resta.Sub(q)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading order lines for the reservation: %w", err)
	}
	if resta.IsPositive() {
		partes = append(partes, parteDaReserva{ref: item, qtd: resta})
	}
	return partes, nil
}

func (r *StockRepositorySQLC) linhasDaOnda(ctx context.Context, empresa, ondaID int64) ([]entity.LinhaOnda, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT w.id, w.item_code, w.mask, w.lot, w.address, COALESCE(a.zone,''), w.pick_sequence, w.quantity, w.picked_qty,
		        l.heat_number, l.expires_at, w.reference_type, w.reference_code
		   FROM stock_picking_wave_lines w
		   JOIN stock_picking_waves o ON o.id=w.wave_id
		   LEFT JOIN stock_lots l ON l.enterprise_id=w.enterprise_id AND l.item_code=w.item_code AND l.mask=w.mask AND l.lot=w.lot
		   LEFT JOIN manufacturing_warehouse_addresses a
		          ON a.enterprise_id=w.enterprise_id AND a.warehouse_id=o.warehouse_id AND a.address=w.address
		  WHERE w.enterprise_id=$1 AND w.wave_id=$2 ORDER BY w.pick_sequence=0, w.pick_sequence, w.address`, empresa, ondaID)
	if err != nil {
		return nil, fmt.Errorf("lendo as linhas da onda: %w", err)
	}
	defer rows.Close()
	out := []entity.LinhaOnda{}
	for rows.Next() {
		var l entity.LinhaOnda
		var qtd, pego decimal.Decimal
		var validade *time.Time
		if err := rows.Scan(&l.ID, &l.ItemCode, &l.Mask, &l.Lot, &l.Address, &l.Zone, &l.PickSequence, &qtd, &pego,
			&l.HeatNumber, &validade, &l.ReferenceType, &l.ReferenceCode); err != nil {
			return nil, fmt.Errorf("lendo linha da onda: %w", err)
		}
		l.Quantity, l.PickedQty, l.ExpiresAt = qtd.InexactFloat64(), pego.InexactFloat64(), validade
		out = append(out, l)
	}
	return out, rows.Err()
}

// ListarOndas devolve as ondas da empresa, as mais recentes primeiro, com as linhas.
func (r *StockRepositorySQLC) ListarOndas(ctx context.Context, status string) ([]*entity.OndaDeSeparacao, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, code, warehouse_id, status, rule, created_at FROM stock_picking_waves
		  WHERE enterprise_id=$1 AND ($2='' OR status=$2) ORDER BY code DESC LIMIT 100`, empresa, strings.ToUpper(strings.TrimSpace(status)))
	if err != nil {
		return nil, fmt.Errorf("listando as ondas: %w", err)
	}
	out := []*entity.OndaDeSeparacao{}
	for rows.Next() {
		o := &entity.OndaDeSeparacao{EmFalta: []entity.FaltaNaOnda{}}
		if err := rows.Scan(&o.ID, &o.Code, &o.WarehouseID, &o.Status, &o.Rule, &o.CreatedAt); err != nil {
			rows.Close()
			return nil, fmt.Errorf("listando as ondas: %w", err)
		}
		out = append(out, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("listando as ondas: %w", err)
	}
	for _, o := range out {
		if o.Linhas, err = r.linhasDaOnda(ctx, empresa, o.ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// LinhasDePedidoParaOnda: pendente de faturar, menos o que está em onda aberta e
// o separado que ainda não virou nota.
func (r *StockRepositorySQLC) LinhasDePedidoParaOnda(ctx context.Context, pedido, almox int64) ([]entity.LinhaPedidoOnda, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`WITH pedido AS (
		   SELECT i.item_code, i.mask, SUM(i.attended_qty) atendido,
		          SUM(GREATEST(i.requested_qty - i.attended_qty - i.cancelled_qty, 0)) pendente
		     FROM sales_order_items i
		     JOIN sales_orders o ON o.code=i.sales_order_code
		     JOIN enterprise e ON e.code=o.enterprise_code
		    WHERE e.id=$1 AND o.code=$2 AND o.is_active AND i.is_active AND i.warehouse_code=$3
		    GROUP BY 1, 2),
		 onda AS (
		   SELECT l.item_code, l.mask,
		          SUM(l.quantity) FILTER (WHERE w.status='ABERTA') aberto,
		          SUM(l.picked_qty) FILTER (WHERE w.status='SEPARADA') separado
		     FROM stock_picking_wave_lines l JOIN stock_picking_waves w ON w.id=l.wave_id
		    WHERE l.enterprise_id=$1 AND l.reference_type=$4 AND l.reference_code=$2
		    GROUP BY 1, 2)
		 SELECT p.item_code, p.mask,
		        p.pendente - COALESCE(o.aberto,0) - GREATEST(COALESCE(o.separado,0) - p.atendido, 0)
		   FROM pedido p LEFT JOIN onda o ON o.item_code=p.item_code AND o.mask=p.mask
		  ORDER BY 1`, empresa, pedido, almox, entity.ReferenceTypeSalesOrder)
	if err != nil {
		return nil, fmt.Errorf("lendo o pedido para a onda: %w", err)
	}
	defer rows.Close()
	out := []entity.LinhaPedidoOnda{}
	for rows.Next() {
		l := entity.LinhaPedidoOnda{ReferenceCode: pedido}
		if err := rows.Scan(&l.ItemCode, &l.Mask, &l.Quantity); err != nil {
			return nil, fmt.Errorf("lendo o pedido para a onda: %w", err)
		}
		if l.Quantity.IsPositive() {
			out = append(out, l)
		}
	}
	return out, rows.Err()
}

// SeparadoParaFaturar lista o que a onda do pedido separou e a nota ainda não levou,
// na ordem da separação: a nota sai no lote e no endereço que foram pegos.
func (r *StockRepositorySQLC) SeparadoParaFaturar(ctx context.Context, pedido, item int64, mask string, almox int64) ([]entity.SeparadoDaOnda, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT l.id, l.lot, l.address, l.picked_qty - l.invoiced_qty
		   FROM stock_picking_wave_lines l JOIN stock_picking_waves w ON w.id=l.wave_id
		  WHERE l.enterprise_id=$1 AND w.status='SEPARADA' AND w.warehouse_id=$5 AND l.reference_type=$2 AND l.reference_code=$3
		    AND l.item_code=$4 AND l.mask=$6 AND l.picked_qty > l.invoiced_qty
		  ORDER BY w.id, l.pick_sequence, l.id`, empresa, entity.ReferenceTypeSalesOrder, pedido, item, almox, mask)
	if err != nil {
		return nil, fmt.Errorf("lendo a separação do pedido: %w", err)
	}
	defer rows.Close()
	out := []entity.SeparadoDaOnda{}
	for rows.Next() {
		var s entity.SeparadoDaOnda
		if err := rows.Scan(&s.Linha, &s.Lote, &s.Endereco, &s.Quantidade); err != nil {
			return nil, fmt.Errorf("lendo a separação do pedido: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// LinhasDeCargaParaOnda junta as linhas dos pedidos da carga: uma onda só para
// tudo que sai no mesmo caminhão.
func (r *StockRepositorySQLC) LinhasDeCargaParaOnda(ctx context.Context, carga, almox int64) ([]entity.LinhaPedidoOnda, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT s.sales_order_code
		   FROM shipment_loads l
		   JOIN shipment_load_shipments ls ON ls.load_id=l.id
		   JOIN shipments s ON s.id=ls.shipment_id
		  WHERE l.enterprise_id=$1 AND l.code=$2 AND s.sales_order_code IS NOT NULL ORDER BY 1`, empresa, carga)
	if err != nil {
		return nil, fmt.Errorf("lendo os pedidos da carga: %w", err)
	}
	var pedidos []int64
	for rows.Next() {
		var p int64
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return nil, fmt.Errorf("lendo os pedidos da carga: %w", err)
		}
		pedidos = append(pedidos, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("lendo os pedidos da carga: %w", err)
	}
	if len(pedidos) == 0 {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("a carga %d não tem pedido nesta empresa", carga))
	}
	out := []entity.LinhaPedidoOnda{}
	for _, p := range pedidos {
		linhas, err := r.LinhasDePedidoParaOnda(ctx, p, almox)
		if err != nil {
			return nil, err
		}
		out = append(out, linhas...)
	}
	return out, nil
}
