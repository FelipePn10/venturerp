package stock

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

// Reserva do componente fabricado para a OF pai (cadeia OF filha → OF pai, ver
// .ai/tasks/cadeia-of-pai-filha.md).
//
// A peça que a OF filha (a bucha do RN-01001) entrega fica reservada para a OF
// pai que ela abastece, na linha exata em que entrou (almoxarifado, endereço,
// lote), até o que falta baixar do item na OF pai. Só a saída de material da
// própria OF pai consome a reserva; a saída de qualquer outro documento não
// leva a peça reservada — libera-se a reserva antes. A OF pai concluída,
// encerrada ou cancelada libera o resto (gatilho da migração 503).

// ReservarParaOFPaiTx reserva para as OFs pai da OF filha o que o movimento de
// entrada trouxe, até disponivel (o que não foi reservado para outra coisa na
// mesma entrega). Pai que começa antes vem primeiro. Só reserva quando o
// material da OF pai é deste almoxarifado (ou não tem almoxarifado). Devolve o
// total reservado.
func ReservarParaOFPaiTx(ctx context.Context, tx pgx.Tx, enterpriseID, childOrderID int64, m *entity.StockMovement,
	disponivel decimal.Decimal, por uuid.UUID) (decimal.Decimal, error) {
	entrou := decimal.Min(entity.SignedQuantity(m.MovementType, m.Quantity), disponivel)
	if !entrou.IsPositive() {
		return decimal.Zero, nil
	}
	rows, err := tx.Query(ctx, `
		SELECT pai.id, pai.order_number, mat.id,
		       GREATEST(mat.quantity - mat.attended_quantity
		                - COALESCE((SELECT SUM(r.quantity) FROM stock_reservations r
		                             WHERE r.enterprise_id = $1 AND r.reference_type = $6
		                               AND r.reference_code = pai.id AND r.reference_item_code = mat.id
		                               AND r.status = 'ACTIVE'), 0), 0)
		  FROM production_orders filha
		  JOIN production_orders pai ON pai.enterprise_id = filha.enterprise_id
		       AND pai.is_active AND pai.status IN ('OPEN', 'RELEASED', 'IN_PROGRESS') AND pai.id <> filha.id
		  JOIN production_order_materials mat ON mat.production_order_id = pai.id AND mat.enterprise_id = pai.enterprise_id
		       AND mat.material_kind = 'DEMAND' AND mat.item_code = $3 AND COALESCE(mat.mask, '') = $4
		       AND (mat.warehouse_id IS NULL OR mat.warehouse_id = $5)
		 WHERE filha.id = $2 AND filha.enterprise_id = $1 AND filha.planned_order_id IS NOT NULL
		   AND EXISTS (SELECT 1 FROM production_chain_links l
		                WHERE l.enterprise_id = filha.enterprise_id AND l.child_planned_order_id = filha.planned_order_id
		                  AND (l.parent_production_order_id = pai.id
		                       OR (l.parent_production_order_id IS NULL AND l.parent_planned_order_id = pai.planned_order_id)))
		 ORDER BY COALESCE(pai.start_date, 'infinity'::date), pai.id, mat.id`,
		enterpriseID, childOrderID, m.ItemCode, m.Mask, m.WarehouseID, entity.ReferenceTypeComponenteDaOF)
	if err != nil {
		return decimal.Zero, fmt.Errorf("lendo as OFs pai da OF %d: %w", childOrderID, err)
	}
	type linhaDoPai struct {
		of, numero, material int64
		falta                decimal.Decimal
	}
	var linhas []linhaDoPai
	for rows.Next() {
		var l linhaDoPai
		if err := rows.Scan(&l.of, &l.numero, &l.material, &l.falta); err != nil {
			rows.Close()
			return decimal.Zero, fmt.Errorf("lendo a OF pai: %w", err)
		}
		if l.falta.IsPositive() {
			linhas = append(linhas, l)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return decimal.Zero, err
	}
	reservado := decimal.Zero
	for _, l := range linhas {
		resta := entrou.Sub(reservado)
		if !resta.IsPositive() {
			break
		}
		material := l.material
		nota := fmt.Sprintf("Componente entregue pela OF %d para a OF pai %d", childOrderID, l.of)
		feito, err := reservarNaLinhaTx(ctx, tx, enterpriseID, entity.ReferenceTypeComponenteDaOF, l.of, &material, m,
			decimal.Min(l.falta, resta), nota, por)
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

// consumirReservaDoComponenteTx: a saída de material da OF pai (requisição,
// baixa, abastecimento da linha) usa primeiro a peça reservada para ela, no
// mesmo almoxarifado. Roda antes de o movimento baixar o saldo.
func consumirReservaDoComponenteTx(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	if m.ReferenceType == nil || m.ReferenceCode == nil || !entity.SignedQuantity(m.MovementType, m.Quantity).IsNegative() {
		return nil
	}
	of, ok, err := ofDonaDaSaida(ctx, tx, enterpriseID, m)
	if err != nil || !ok {
		return err
	}
	item, mask, almox := m.ItemCode, m.Mask, m.WarehouseID
	q := m.Quantity.Abs()
	_, err = liberarReservasTx(ctx, tx, enterpriseID, entity.ReferenceTypeComponenteDaOF, of,
		filtroDaReserva{almox: &almox, item: &item, mask: &mask}, &q, true)
	return err
}

// ofDonaDaSaida: a OF cuja reserva a saída pode usar — a própria OF, ou a OF da
// OS de terceiro na remessa (a peça vai ao terceiro para a etapa dessa OF; antes
// a reserva barrava a remessa e era preciso liberá-la à mão).
func ofDonaDaSaida(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) (int64, bool, error) {
	switch *m.ReferenceType {
	case entity.ReferenceTypeProductionOrder:
		return *m.ReferenceCode, true, nil
	case entity.ReferenceTypeOSTerceiro:
		return ofDaOSTx(ctx, tx, enterpriseID, *m.ReferenceCode)
	}
	return 0, false, nil
}

func ofDaOSTx(ctx context.Context, tx pgx.Tx, enterpriseID, osCode int64) (int64, bool, error) {
	var of *int64
	err := tx.QueryRow(ctx, `SELECT production_order_id FROM third_party_service_orders WHERE enterprise_id = $1 AND code = $2`,
		enterpriseID, osCode).Scan(&of)
	if err == pgx.ErrNoRows || (err == nil && of == nil) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("lendo a OF da OS de terceiro %d: %w", osCode, err)
	}
	return *of, true, nil
}

// reservarRetornoDaOSTx: a peça que volta do terceiro para a fábrica fica de
// novo reservada para a OF da OS, na linha de material dela, até o que falta
// baixar. A entrada no almoxarifado do terceiro (a remessa) não reserva.
func reservarRetornoDaOSTx(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	if m.ReferenceType == nil || *m.ReferenceType != entity.ReferenceTypeOSTerceiro || m.ReferenceCode == nil ||
		!entity.SignedQuantity(m.MovementType, m.Quantity).IsPositive() {
		return nil
	}
	var doTerceiro bool
	if err := tx.QueryRow(ctx, `SELECT supplier_code IS NOT NULL FROM warehouse WHERE id = $1 AND enterprise_id = $2`,
		m.WarehouseID, enterpriseID).Scan(&doTerceiro); err != nil || doTerceiro {
		return err
	}
	of, ok, err := ofDaOSTx(ctx, tx, enterpriseID, *m.ReferenceCode)
	if err != nil || !ok {
		return err
	}
	rows, err := tx.Query(ctx, `
		SELECT mat.id, GREATEST(mat.quantity - mat.attended_quantity
		         - COALESCE((SELECT SUM(r.quantity) FROM stock_reservations r
		                      WHERE r.enterprise_id = $1 AND r.reference_type = $6 AND r.reference_code = o.id
		                        AND r.reference_item_code = mat.id AND r.status = 'ACTIVE'), 0), 0)
		  FROM production_orders o
		  JOIN production_order_materials mat ON mat.production_order_id = o.id AND mat.enterprise_id = o.enterprise_id
		       AND mat.material_kind = 'DEMAND' AND mat.item_code = $3 AND COALESCE(mat.mask, '') = $4
		 WHERE o.id = $2 AND o.enterprise_id = $1 AND o.is_active AND o.status IN ('OPEN', 'RELEASED', 'IN_PROGRESS')
		 -- A linha do almoxarifado em que a peça voltou primeiro; voltando a outro,
		 -- a peça segue reservada para a OF (é dela) até ser levada à linha.
		 ORDER BY (mat.warehouse_id IS NOT DISTINCT FROM $5) DESC, mat.id`, enterpriseID, of, m.ItemCode, m.Mask, m.WarehouseID, entity.ReferenceTypeComponenteDaOF)
	if err != nil {
		return fmt.Errorf("lendo o material da OF %d: %w", of, err)
	}
	type linha struct {
		id    int64
		falta decimal.Decimal
	}
	var linhas []linha
	for rows.Next() {
		var l linha
		if err := rows.Scan(&l.id, &l.falta); err != nil {
			rows.Close()
			return err
		}
		linhas = append(linhas, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	resta := m.Quantity.Abs()
	for _, l := range linhas {
		if !resta.IsPositive() {
			break
		}
		material := l.id
		feito, err := reservarNaLinhaTx(ctx, tx, enterpriseID, entity.ReferenceTypeComponenteDaOF, of, &material, m,
			decimal.Min(l.falta, resta), fmt.Sprintf("Peça de volta do terceiro (OS %d) para a OF %d", *m.ReferenceCode, of), m.CreatedBy)
		if err != nil {
			return err
		}
		resta = resta.Sub(feito)
	}
	return nil
}

// conferirReservaDeOutraOFTx recusa a saída que deixaria no almoxarifado menos
// do que está reservado para OFs pai: a peça é de outro pedido. Roda depois de
// o movimento baixar o saldo (e depois de a própria OF pai consumir a reserva
// dela). Saídas liberadas (inventário, estorno, devolução de compra…) passam,
// e a reserva cai até o saldo que sobrou: a peça que saiu não existe mais.
func conferirReservaDeOutraOFTx(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	if !entity.SignedQuantity(m.MovementType, movementQuantity(m)).IsNegative() {
		return nil
	}
	if saidaLiberada(m) {
		return apararReservaDoComponenteTx(ctx, tx, enterpriseID, m)
	}
	var saldo, reservado decimal.Decimal
	var ordens string
	err := tx.QueryRow(ctx, `
		SELECT COALESCE((SELECT SUM(b.quantity) FROM stock_balances b
		                  WHERE b.enterprise_id = $1 AND b.item_code = $2 AND b.mask = $3 AND b.warehouse_id = $4), 0),
		       COALESCE(SUM(r.quantity), 0),
		       COALESCE(string_agg(DISTINCT o.id::text, ', '), '')
		  FROM stock_reservations r
		  JOIN production_orders o ON o.id = r.reference_code AND o.enterprise_id = r.enterprise_id
		 WHERE r.enterprise_id = $1 AND r.item_code = $2 AND r.mask = $3 AND r.warehouse_id = $4
		   AND r.reference_type = $5 AND r.status = 'ACTIVE'`,
		enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, entity.ReferenceTypeComponenteDaOF).Scan(&saldo, &reservado, &ordens)
	if err != nil {
		return fmt.Errorf("conferindo a reserva das OFs pai: %w", err)
	}
	if !reservado.IsPositive() || !saldo.LessThan(reservado) {
		return nil
	}
	return errorsuc.NewValidationError(fmt.Sprintf(
		"o saldo do item %d neste almoxarifado está reservado para a OF %s (peça fabricada para ela pela OF filha): "+
			"a saída deixaria %s, abaixo dos %s reservados. Libere a reserva antes de usar a peça em outro documento",
		m.ItemCode, ordens, entity.QuantidadeBR(saldo), entity.QuantidadeBR(reservado)))
}

// apararReservaDoComponenteTx libera a reserva para OF pai que passou do saldo
// do almoxarifado (a saída liberada levou a peça): da OF pai que começa mais
// tarde para a que começa antes, até a reserva caber no saldo.
func apararReservaDoComponenteTx(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	rows, err := tx.Query(ctx, `
		SELECT r.reference_code, SUM(r.quantity),
		       GREATEST(COALESCE((SELECT SUM(b.quantity) FROM stock_balances b
		                           WHERE b.enterprise_id = $1 AND b.item_code = $2 AND b.mask = $3 AND b.warehouse_id = $4), 0), 0)
		  FROM stock_reservations r
		  LEFT JOIN production_orders o ON o.id = r.reference_code AND o.enterprise_id = r.enterprise_id
		 WHERE r.enterprise_id = $1 AND r.item_code = $2 AND r.mask = $3 AND r.warehouse_id = $4
		   AND r.reference_type = $5 AND r.status = 'ACTIVE'
		 GROUP BY r.reference_code, o.start_date
		 ORDER BY o.start_date DESC NULLS FIRST, r.reference_code DESC`,
		enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, entity.ReferenceTypeComponenteDaOF)
	if err != nil {
		return fmt.Errorf("lendo a reserva das OFs pai: %w", err)
	}
	type reservaDaOF struct {
		of  int64
		qtd decimal.Decimal
	}
	var reservas []reservaDaOF
	total, saldo := decimal.Zero, decimal.Zero
	for rows.Next() {
		var r reservaDaOF
		if err := rows.Scan(&r.of, &r.qtd, &saldo); err != nil {
			rows.Close()
			return fmt.Errorf("lendo a reserva das OFs pai: %w", err)
		}
		reservas = append(reservas, r)
		total = total.Add(r.qtd)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	excesso := total.Sub(saldo)
	item, mask, almox := m.ItemCode, m.Mask, m.WarehouseID
	for _, r := range reservas {
		if !excesso.IsPositive() {
			break
		}
		q := decimal.Min(excesso, r.qtd)
		liberado, err := liberarReservasTx(ctx, tx, enterpriseID, entity.ReferenceTypeComponenteDaOF, r.of,
			filtroDaReserva{almox: &almox, item: &item, mask: &mask}, &q, false)
		if err != nil {
			return err
		}
		excesso = excesso.Sub(liberado)
	}
	return nil
}

// ReservarEntregasDasFilhasTx: a OF pai que nasce DEPOIS da entrega da OF
// filha (firmar de baixo para cima — a bucha é feita antes de o RN ser
// firmado) reserva, ao ser criada, o que as filhas dela já entregaram e ainda
// está livre no estoque. Na entrega não havia OF pai para reservar, e sem isto
// a peça ficava livre para a OF de outro pedido.
//
// O teto de cada material é o menor entre o que falta baixar na OF pai e o que
// as filhas ligadas a ela entregaram menos o que já foi reservado (ou
// consumido pela reserva) para as outras OFs pai dessas filhas — uma bucha em
// lote agrupado abastece vários RN e cada um leva só a sua parte. A reserva vai
// na linha (almoxarifado, endereço, lote) das entregas, mais nova primeiro, e
// nunca passa do saldo livre da linha. Devolve o total reservado.
func ReservarEntregasDasFilhasTx(ctx context.Context, tx pgx.Tx, enterpriseID, parentOrderID int64, por uuid.UUID) (decimal.Decimal, error) {
	rows, err := tx.Query(ctx, `
		WITH filhas AS (
		    SELECT DISTINCT filha.id, filha.item_code, COALESCE(filha.mask, '') AS mask, filha.planned_order_id
		      FROM production_orders pai
		      JOIN production_chain_links l ON l.enterprise_id = pai.enterprise_id
		           AND (l.parent_production_order_id = pai.id
		                OR (l.parent_production_order_id IS NULL AND l.parent_planned_order_id = pai.planned_order_id))
		      JOIN production_orders filha ON filha.enterprise_id = l.enterprise_id
		           AND filha.planned_order_id = l.child_planned_order_id AND filha.id <> pai.id
		     WHERE pai.id = $2 AND pai.enterprise_id = $1 AND pai.planned_order_id IS NOT NULL
		       AND pai.status IN ('OPEN', 'RELEASED', 'IN_PROGRESS')
		), entregue AS (
		    SELECT f.item_code, f.mask, SUM(m.quantity) AS qtd
		      FROM filhas f
		      JOIN stock_movements m ON m.enterprise_id = $1 AND m.reference_type = 'PRODUCTION_ORDER'
		           AND m.reference_code = f.id AND m.item_code = f.item_code AND COALESCE(m.mask, '') = f.mask
		           AND m.movement_type IN ('EP', 'EPP', 'EPE') AND m.reversed_by_movement_id IS NULL
		     GROUP BY f.item_code, f.mask
		), outros_pais AS (
		    SELECT DISTINCT f.item_code, f.mask, outro.id AS of
		      FROM filhas f
		      JOIN production_chain_links l ON l.enterprise_id = $1 AND l.child_planned_order_id = f.planned_order_id
		      JOIN production_orders outro ON outro.enterprise_id = $1 AND outro.id <> $2
		           AND (outro.id = l.parent_production_order_id
		                OR (l.parent_production_order_id IS NULL AND outro.planned_order_id = l.parent_planned_order_id))
		), ja_dado AS (
		    SELECT o.item_code, o.mask, SUM(r.quantity) AS qtd
		      FROM outros_pais o
		      JOIN stock_reservations r ON r.enterprise_id = $1 AND r.reference_type = $3
		           AND r.reference_code = o.of AND r.item_code = o.item_code AND COALESCE(r.mask, '') = o.mask
		           AND r.status IN ('ACTIVE', 'CONSUMED')
		     GROUP BY o.item_code, o.mask
		)
		SELECT mat.id, mat.item_code, COALESCE(mat.mask, ''), mat.warehouse_id,
		       LEAST(GREATEST(mat.quantity - mat.attended_quantity, 0),
		             GREATEST(e.qtd - COALESCE(j.qtd, 0), 0))
		  FROM production_order_materials mat
		  JOIN entregue e ON e.item_code = mat.item_code AND e.mask = COALESCE(mat.mask, '')
		  LEFT JOIN ja_dado j ON j.item_code = e.item_code AND j.mask = e.mask
		 WHERE mat.production_order_id = $2 AND mat.enterprise_id = $1 AND mat.material_kind = 'DEMAND'
		 ORDER BY mat.id`, enterpriseID, parentOrderID, entity.ReferenceTypeComponenteDaOF)
	if err != nil {
		return decimal.Zero, fmt.Errorf("lendo o que as OFs filha da OF %d já entregaram: %w", parentOrderID, err)
	}
	type linhaDoPai struct {
		material, item int64
		mask           string
		almox          *int64
		teto           decimal.Decimal
	}
	var linhas []linhaDoPai
	for rows.Next() {
		var l linhaDoPai
		if err := rows.Scan(&l.material, &l.item, &l.mask, &l.almox, &l.teto); err != nil {
			rows.Close()
			return decimal.Zero, fmt.Errorf("lendo o material da OF pai: %w", err)
		}
		if l.teto.IsPositive() {
			linhas = append(linhas, l)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return decimal.Zero, err
	}
	total := decimal.Zero
	for _, l := range linhas {
		// As linhas das entregas das filhas, mais nova primeiro.
		ent, err := tx.Query(ctx, `
			SELECT m.warehouse_id, m.address, m.lot, SUM(m.quantity)
			  FROM production_orders pai
			  JOIN production_chain_links lk ON lk.enterprise_id = pai.enterprise_id
			       AND (lk.parent_production_order_id = pai.id
			            OR (lk.parent_production_order_id IS NULL AND lk.parent_planned_order_id = pai.planned_order_id))
			  JOIN production_orders filha ON filha.enterprise_id = pai.enterprise_id AND filha.planned_order_id = lk.child_planned_order_id
			  JOIN stock_movements m ON m.enterprise_id = pai.enterprise_id AND m.reference_type = 'PRODUCTION_ORDER'
			       AND m.reference_code = filha.id AND m.item_code = $3 AND COALESCE(m.mask, '') = $4
			       AND m.movement_type IN ('EP', 'EPP', 'EPE') AND m.reversed_by_movement_id IS NULL
			 WHERE pai.id = $2 AND pai.enterprise_id = $1 AND ($5::bigint IS NULL OR m.warehouse_id = $5)
			 GROUP BY m.warehouse_id, m.address, m.lot
			 ORDER BY MAX(m.id) DESC`, enterpriseID, parentOrderID, l.item, l.mask, l.almox)
		if err != nil {
			return total, fmt.Errorf("lendo as entregas do componente: %w", err)
		}
		type entrega struct {
			mov entity.StockMovement
			qtd decimal.Decimal
		}
		var entregas []entrega
		for ent.Next() {
			e := entrega{mov: entity.StockMovement{ItemCode: l.item, Mask: l.mask}}
			if err := ent.Scan(&e.mov.WarehouseID, &e.mov.Address, &e.mov.Lot, &e.qtd); err != nil {
				ent.Close()
				return total, fmt.Errorf("lendo a entrega do componente: %w", err)
			}
			entregas = append(entregas, e)
		}
		ent.Close()
		if err := ent.Err(); err != nil {
			return total, err
		}
		resta := l.teto
		material := l.material
		for i := range entregas {
			if !resta.IsPositive() {
				break
			}
			nota := fmt.Sprintf("Componente já entregue pela OF filha, reservado ao criar a OF pai %d", parentOrderID)
			feito, err := reservarNaLinhaTx(ctx, tx, enterpriseID, entity.ReferenceTypeComponenteDaOF, parentOrderID, &material,
				&entregas[i].mov, decimal.Min(resta, entregas[i].qtd), nota, por)
			if err != nil {
				return total, err
			}
			resta = resta.Sub(feito)
			total = total.Add(feito)
		}
	}
	return total, nil
}
