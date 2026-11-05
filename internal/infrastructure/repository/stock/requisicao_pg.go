package stock

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

const requisicaoCols = `id, code, warehouse_id, requester_name, requester_employee_code, cost_center_code, needed_date,
	kind, destination_warehouse_id, status, notes, created_by, created_at, updated_at`

func scanRequisicao(row pgx.Row) (*entity.RequisicaoMaterial, error) {
	var r entity.RequisicaoMaterial
	if err := row.Scan(&r.ID, &r.Code, &r.WarehouseID, &r.RequesterName, &r.RequesterEmployeeCode, &r.CostCenterCode,
		&r.NeededDate, &r.Kind, &r.DestinoID, &r.Status, &r.Notes, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

// CriarRequisicaoMaterial grava a requisição com as linhas. Sem código, usa o
// próximo da empresa.
func (r *StockRepositorySQLC) CriarRequisicaoMaterial(ctx context.Context, req *entity.RequisicaoMaterial) (*entity.RequisicaoMaterial, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning requisition tx: %w", err)
	}
	defer tx.Rollback(ctx)
	almoxes := []int64{req.WarehouseID}
	if req.DestinoID != nil {
		if *req.DestinoID == req.WarehouseID {
			return nil, errorsuc.NewValidationError("o almoxarifado de destino não pode ser o que atende")
		}
		almoxes = append(almoxes, *req.DestinoID)
	}
	var n int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM warehouse WHERE id = ANY($1) AND enterprise_id=$2`, almoxes, empresa).Scan(&n); err != nil {
		return nil, fmt.Errorf("checking requisition warehouse: %w", err)
	}
	if n != len(almoxes) {
		return nil, errorsuc.NewValidationError("almoxarifado da requisição não existe nesta empresa")
	}
	kind, status := entity.RequisicaoConsumo, entity.RequisicaoAberta
	if req.DestinoID != nil {
		kind = entity.RequisicaoTransferencia
	}
	if req.AguardaAprovacao {
		status = entity.RequisicaoPendente
	}
	codes := make([]int64, 0, len(req.Itens))
	for _, it := range req.Itens {
		codes = append(codes, it.ItemCode)
	}
	var achados int
	if err := tx.QueryRow(ctx, `SELECT COUNT(DISTINCT code) FROM items WHERE enterprise_id=$1 AND code = ANY($2)`, empresa, codes).Scan(&achados); err != nil {
		return nil, fmt.Errorf("checking requisition items: %w", err)
	}
	distintos := map[int64]bool{}
	for _, c := range codes {
		distintos[c] = true
	}
	if achados != len(distintos) {
		return nil, errorsuc.NewValidationError("há item na requisição que não existe nesta empresa")
	}
	if err := TravarItensTx(ctx, tx, empresa, codes...); err != nil {
		return nil, err
	}
	if req.Code <= 0 {
		// A trava serializa a numeração: duas requisições ao mesmo tempo tirariam o mesmo código.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('material_requisition_code', $1))`, empresa); err != nil {
			return nil, fmt.Errorf("locking requisition numbering: %w", err)
		}
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(code),0)+1 FROM material_requisitions WHERE enterprise_id=$1`, empresa).Scan(&req.Code); err != nil {
			return nil, fmt.Errorf("next requisition code: %w", err)
		}
	}
	criada, err := scanRequisicao(tx.QueryRow(ctx,
		`INSERT INTO material_requisitions (enterprise_id, code, warehouse_id, requester_name, requester_employee_code,
		     cost_center_code, needed_date, kind, destination_warehouse_id, status, notes, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING `+requisicaoCols,
		empresa, req.Code, req.WarehouseID, req.RequesterName, req.RequesterEmployeeCode, req.CostCenterCode,
		req.NeededDate, kind, req.DestinoID, status, req.Notes, req.CreatedBy))
	if err != nil {
		return nil, fmt.Errorf("creating material requisition: %w", err)
	}
	for i, it := range req.Itens {
		var id int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO material_requisition_items (requisition_id, sequence, item_code, mask, quantity, notes)
			 VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`, criada.ID, i+1, it.ItemCode, it.Mask, it.Quantity, it.Notes).Scan(&id); err != nil {
			return nil, fmt.Errorf("creating requisition line %d: %w", i+1, err)
		}
		it.ID, it.Sequence = id, i+1
		criada.Itens = append(criada.Itens, it)
	}
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(SUM(l.quantity * COALESCE(c.custo, 0)), 0)
		   FROM material_requisition_items l
		   LEFT JOIN LATERAL (SELECT MAX(b.avg_cost) custo FROM stock_balances b
		                       WHERE b.enterprise_id=$2 AND b.item_code=l.item_code AND b.mask=l.mask AND b.warehouse_id=$3) c ON TRUE
		  WHERE l.requisition_id=$1`, criada.ID, empresa, req.WarehouseID).Scan(&criada.Valor); err != nil {
		return nil, fmt.Errorf("valuing requisition: %w", err)
	}
	if criada.Status == entity.RequisicaoAberta {
		if err := reservarRequisicaoTx(ctx, tx, empresa, criada); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing requisition: %w", err)
	}
	return criada, nil
}

func (r *StockRepositorySQLC) itensDaRequisicao(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, id int64) ([]entity.RequisicaoMaterialItem, error) {
	return lerItensDaRequisicao(ctx, q, id)
}

func lerItensDaRequisicao(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, id int64) ([]entity.RequisicaoMaterialItem, error) {
	rows, err := q.Query(ctx,
		`SELECT l.id, l.sequence, l.item_code, l.mask, l.quantity, l.issued_qty, l.notes,
		        COALESCE((SELECT SUM(s.quantity) FROM stock_reservations s
		                   WHERE s.reference_type=$2 AND s.reference_item_code=l.id AND s.status='ACTIVE'), 0)
		   FROM material_requisition_items l WHERE l.requisition_id=$1 ORDER BY l.sequence`, id, entity.ReferenceTypeRequisition)
	if err != nil {
		return nil, fmt.Errorf("listing requisition lines: %w", err)
	}
	defer rows.Close()
	var out []entity.RequisicaoMaterialItem
	for rows.Next() {
		var it entity.RequisicaoMaterialItem
		if err := rows.Scan(&it.ID, &it.Sequence, &it.ItemCode, &it.Mask, &it.Quantity, &it.IssuedQty, &it.Notes, &it.ReservedQty); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ObterRequisicaoMaterial lê a requisição da empresa com as linhas.
func (r *StockRepositorySQLC) ObterRequisicaoMaterial(ctx context.Context, id int64) (*entity.RequisicaoMaterial, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	req, err := scanRequisicao(r.pool.QueryRow(ctx, `SELECT `+requisicaoCols+` FROM material_requisitions WHERE id=$1 AND enterprise_id=$2`, id, empresa))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("requisição %d não encontrada", id))
	}
	if err != nil {
		return nil, fmt.Errorf("reading material requisition: %w", err)
	}
	if req.Itens, err = r.itensDaRequisicao(ctx, r.pool, id); err != nil {
		return nil, err
	}
	return req, nil
}

// ListarRequisicoesMaterial lista as requisições da empresa (filtro opcional
// de situação), as mais recentes primeiro, com as linhas.
func (r *StockRepositorySQLC) ListarRequisicoesMaterial(ctx context.Context, status *string) ([]*entity.RequisicaoMaterial, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+requisicaoCols+` FROM material_requisitions
		  WHERE enterprise_id=$1 AND ($2::text IS NULL OR status=$2) ORDER BY code DESC LIMIT 500`, empresa, status)
	if err != nil {
		return nil, fmt.Errorf("listing material requisitions: %w", err)
	}
	var out []*entity.RequisicaoMaterial
	for rows.Next() {
		req, err := scanRequisicao(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, req)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, req := range out {
		if req.Itens, err = r.itensDaRequisicao(ctx, r.pool, req.ID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// AtenderRequisicaoMaterial lança a saída de cada linha atendida no centro de
// custo da requisição e atualiza o atendido e a situação, numa transação. Não
// entrega mais do que o pendente da linha.
func (r *StockRepositorySQLC) AtenderRequisicaoMaterial(ctx context.Context, id int64, linhas []entity.AtendimentoLinha, ator uuid.UUID) (*entity.RequisicaoMaterial, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning issue tx: %w", err)
	}
	defer tx.Rollback(ctx)
	req, err := scanRequisicao(tx.QueryRow(ctx, `SELECT `+requisicaoCols+` FROM material_requisitions WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, id, empresa))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("requisição %d não encontrada", id))
	}
	if err != nil {
		return nil, fmt.Errorf("locking material requisition: %w", err)
	}
	if !req.Atendivel() {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a requisição %d está %s: não há o que atender", req.Code, rotuloRequisicao(req.Status)))
	}
	itens, err := r.itensDaRequisicao(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if err := travarItensDaRequisicaoTx(ctx, tx, empresa, itens); err != nil {
		return nil, err
	}
	porID := map[int64]*entity.RequisicaoMaterialItem{}
	for i := range itens {
		porID[itens[i].ID] = &itens[i]
	}
	ref := entity.ReferenceTypeRequisition
	codigo := req.Code
	nota := fmt.Sprintf("Requisição ao almoxarifado %d", req.Code)
	for _, l := range linhas {
		it, ok := porID[l.ItemID]
		if !ok {
			return nil, errorsuc.NewValidationError("uma das linhas informadas não é desta requisição")
		}
		if !l.Quantity.IsPositive() {
			continue
		}
		if l.Quantity.GreaterThan(it.Pendente()) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("linha %d: a entrega de %s passa do pendente, que é %s", it.Sequence, entity.QuantidadeBR(l.Quantity), entity.QuantidadeBR(it.Pendente())))
		}
		// A reserva é desta entrega: solta antes, senão a saída a vê como de outro.
		if err := consumirReservaDaLinhaTx(ctx, tx, empresa, it.ID, l.Quantity); err != nil {
			return nil, err
		}
		if req.Kind != entity.RequisicaoTransferencia && l.ViaTransito {
			return nil, errorsuc.NewValidationError("só a requisição de transferência vai pelo trânsito")
		}
		if l.ViaTransito {
			_, err = enviarParaTransitoTx(ctx, tx, empresa, entity.EnvioTransito{
				ItemCode: it.ItemCode, Mask: it.Mask, Lote: l.Lot, EnderecoDe: l.Address, DeAlmox: req.WarehouseID,
				AlmoxTransito: l.AlmoxTransito, ParaAlmox: *req.DestinoID, Quantidade: l.Quantity, Notas: &nota, Ator: ator, RequisicaoID: &req.ID,
			})
		} else if req.Kind == entity.RequisicaoTransferencia {
			_, err = transferirTx(ctx, tx, empresa, entity.TransferenciaEstoque{
				ItemCode: it.ItemCode, Mask: it.Mask, DeAlmox: req.WarehouseID, ParaAlmox: *req.DestinoID, Quantidade: l.Quantity,
				Lote: l.Lot, EnderecoDe: l.Address, EnderecoPara: l.DestinationAddress, Notas: &nota, Ator: ator,
				ReferenciaTipo: &ref, ReferenciaCodigo: &codigo,
			}, true)
		} else {
			err = CreateMovementTx(ctx, tx, empresa, &entity.StockMovement{
				ItemCode: it.ItemCode, Mask: it.Mask, WarehouseID: req.WarehouseID, MovementType: entity.MovementTypeOut,
				Quantity: l.Quantity, Lot: l.Lot, Address: l.Address,
				ReferenceType: &ref, ReferenceCode: &codigo, Notes: &nota, CreatedBy: ator, CostCenterCode: req.CostCenterCode,
				ExigirRastreio: true,
			})
		}
		if err != nil {
			return nil, fmt.Errorf("linha %d: %w", it.Sequence, err)
		}
		if _, err := tx.Exec(ctx, `UPDATE material_requisition_items SET issued_qty = issued_qty + $2 WHERE id=$1`, it.ID, l.Quantity); err != nil {
			return nil, fmt.Errorf("updating issued quantity: %w", err)
		}
		it.IssuedQty = it.IssuedQty.Add(l.Quantity)
		it.ReservedQty = decimal.Max(it.ReservedQty.Sub(l.Quantity), decimal.Zero)
	}
	status := entity.RequisicaoAtendida
	algum := false
	for _, it := range itens {
		if it.Pendente().IsPositive() {
			status = entity.RequisicaoParcial
		}
		if it.IssuedQty.IsPositive() {
			algum = true
		}
	}
	if !algum {
		status = entity.RequisicaoAberta
	}
	if _, err := tx.Exec(ctx, `UPDATE material_requisitions SET status=$2, updated_at=NOW() WHERE id=$1`, id, status); err != nil {
		return nil, fmt.Errorf("updating requisition status: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing issue: %w", err)
	}
	req.Status, req.Itens = status, itens
	return req, nil
}

// CancelarRequisicaoMaterial encerra o que falta atender e devolve a reserva. O
// já entregue fica.
func (r *StockRepositorySQLC) CancelarRequisicaoMaterial(ctx context.Context, id int64) error {
	return r.encerrarRequisicao(ctx, id, entity.RequisicaoCancelada, func(s string) error {
		if s == entity.RequisicaoAtendida || s == entity.RequisicaoCancelada || s == entity.RequisicaoReprovada {
			return errorsuc.NewValidationError(fmt.Sprintf("a requisição está %s: não há o que cancelar", rotuloRequisicao(s)))
		}
		return nil
	})
}

// ReprovarRequisicaoMaterial aplica a decisão da alçada. Já decidida não muda:
// quem cancelou enquanto a aprovação corria não vira reprovada.
func (r *StockRepositorySQLC) ReprovarRequisicaoMaterial(ctx context.Context, id int64) error {
	err := r.encerrarRequisicao(ctx, id, entity.RequisicaoReprovada, func(s string) error {
		if s != entity.RequisicaoPendente {
			return errRequisicaoJaDecidida
		}
		return nil
	})
	if errors.Is(err, errRequisicaoJaDecidida) {
		return nil
	}
	return err
}

var errRequisicaoJaDecidida = errors.New("requisição já decidida")

func (r *StockRepositorySQLC) encerrarRequisicao(ctx context.Context, id int64, status string, conferir func(string) error) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning requisition close tx: %w", err)
	}
	defer tx.Rollback(ctx)
	var atual string
	err = tx.QueryRow(ctx, `SELECT status FROM material_requisitions WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, id, empresa).Scan(&atual)
	if err == nil {
		itens, e := r.itensDaRequisicao(ctx, tx, id)
		if e != nil {
			return e
		}
		if e := travarItensDaRequisicaoTx(ctx, tx, empresa, itens); e != nil {
			return e
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("requisição %d não encontrada", id))
	}
	if err != nil {
		return fmt.Errorf("reading requisition status: %w", err)
	}
	if err := conferir(atual); err != nil {
		return err
	}
	if err := soltarReservasDaRequisicaoTx(ctx, tx, empresa, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE material_requisitions SET status=$3, updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, id, empresa, status); err != nil {
		return fmt.Errorf("closing requisition: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing requisition close: %w", err)
	}
	return nil
}

// LiberarRequisicaoAprovada abre a requisição aprovada e reserva o material.
// Idempotente: requisição que não está pendente não muda.
func (r *StockRepositorySQLC) LiberarRequisicaoAprovada(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning requisition release tx: %w", err)
	}
	defer tx.Rollback(ctx)
	req, err := scanRequisicao(tx.QueryRow(ctx, `SELECT `+requisicaoCols+` FROM material_requisitions WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, id, empresa))
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("requisição %d não encontrada", id))
	}
	if err != nil {
		return fmt.Errorf("locking material requisition: %w", err)
	}
	if req.Status != entity.RequisicaoPendente {
		return nil
	}
	if req.Itens, err = r.itensDaRequisicao(ctx, tx, id); err != nil {
		return err
	}
	if err := travarItensDaRequisicaoTx(ctx, tx, empresa, req.Itens); err != nil {
		return err
	}
	if err := reservarRequisicaoTx(ctx, tx, empresa, req); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE material_requisitions SET status='ABERTA', updated_at=NOW() WHERE id=$1`, id); err != nil {
		return fmt.Errorf("opening requisition: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing requisition release: %w", err)
	}
	return nil
}

// reservarRequisicaoTx reserva o pendente de cada linha, até o que estiver livre.
// Falta de saldo não impede abrir: o que não coube fica pendente sem reserva e
// aparece na reposição. Almoxarifado que não aceita reserva só não reserva.
func reservarRequisicaoTx(ctx context.Context, tx pgx.Tx, empresa int64, req *entity.RequisicaoMaterial) error {
	var aceita bool
	if err := tx.QueryRow(ctx, `SELECT reservations_allowed FROM warehouse WHERE id=$1 AND enterprise_id=$2`, req.WarehouseID, empresa).Scan(&aceita); err != nil {
		return fmt.Errorf("reading warehouse reservation flag: %w", err)
	}
	if !aceita {
		return nil
	}
	nota := fmt.Sprintf("Requisição ao almoxarifado %d", req.Code)
	for i := range req.Itens {
		it := &req.Itens[i]
		var livre decimal.Decimal
		err := tx.QueryRow(ctx,
			`WITH saldo AS (SELECT quantity, reserved_qty FROM stock_balances
			                 WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 FOR UPDATE)
			 SELECT COALESCE((SELECT SUM(GREATEST(quantity - reserved_qty, 0)) FROM saldo), 0)
			      - COALESCE((SELECT SUM(b.quantity) FROM stock_lot_balances b
			                   WHERE b.enterprise_id=$1 AND b.item_code=$2 AND b.mask=$3 AND b.warehouse_id=$4
			                     AND b.quantity > 0 AND `+loteSemSaidaSQL+`), 0)`,
			empresa, it.ItemCode, it.Mask, req.WarehouseID).Scan(&livre)
		if err != nil {
			return fmt.Errorf("reading free balance of line %d: %w", it.Sequence, err)
		}
		q := decimal.Min(it.Pendente().Sub(it.ReservedQty), livre)
		if !q.IsPositive() {
			continue
		}
		// Uma reserva por linha: é ela que a entrega consome.
		tag, err := tx.Exec(ctx,
			`UPDATE stock_reservations SET quantity = quantity + $4, updated_at=NOW()
			  WHERE enterprise_id=$1 AND reference_type=$2 AND reference_item_code=$3 AND status='ACTIVE'`,
			empresa, entity.ReferenceTypeRequisition, it.ID, q)
		if err != nil {
			return fmt.Errorf("reserving line %d: %w", it.Sequence, err)
		}
		if tag.RowsAffected() == 0 {
			if _, err := tx.Exec(ctx,
				`INSERT INTO stock_reservations (item_code, mask, warehouse_id, quantity, reference_type, reference_code,
				     reference_item_code, status, notes, created_by, enterprise_id)
				 VALUES ($1,$2,$3,$4,$5,$6,$7,'ACTIVE',$8,$9,$10)`,
				it.ItemCode, it.Mask, req.WarehouseID, q, entity.ReferenceTypeRequisition, req.Code, it.ID, nota, req.CreatedBy, empresa); err != nil {
				return fmt.Errorf("reserving line %d: %w", it.Sequence, err)
			}
		}
		if err := adjustReservedTx(ctx, tx, empresa, it.ItemCode, it.Mask, req.WarehouseID, q); err != nil {
			return err
		}
		it.ReservedQty = it.ReservedQty.Add(q)
	}
	return nil
}

func travarItensDaRequisicaoTx(ctx context.Context, tx pgx.Tx, empresa int64, itens []entity.RequisicaoMaterialItem) error {
	codigos := make([]int64, 0, len(itens))
	for _, it := range itens {
		codigos = append(codigos, it.ItemCode)
	}
	return TravarItensTx(ctx, tx, empresa, codigos...)
}

// consumirReservaDaLinhaTx baixa da reserva da linha o que está sendo entregue.
func consumirReservaDaLinhaTx(ctx context.Context, tx pgx.Tx, empresa, linha int64, qtd decimal.Decimal) error {
	var id, item, almox int64
	var mask string
	var reservado decimal.Decimal
	err := tx.QueryRow(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity FROM stock_reservations
		  WHERE enterprise_id=$1 AND reference_type=$2 AND reference_item_code=$3 AND status='ACTIVE' FOR UPDATE`,
		empresa, entity.ReferenceTypeRequisition, linha).Scan(&id, &item, &mask, &almox, &reservado)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading line reservation: %w", err)
	}
	consumido := decimal.Min(qtd, reservado)
	if consumido.Equal(reservado) {
		_, err = tx.Exec(ctx, `UPDATE stock_reservations SET status='CONSUMED', updated_at=NOW() WHERE id=$1`, id)
	} else {
		_, err = tx.Exec(ctx, `UPDATE stock_reservations SET quantity = quantity - $2, updated_at=NOW() WHERE id=$1`, id, consumido)
	}
	if err != nil {
		return fmt.Errorf("consuming line reservation: %w", err)
	}
	return adjustReservedTx(ctx, tx, empresa, item, mask, almox, consumido.Neg())
}

func soltarReservasDaRequisicaoTx(ctx context.Context, tx pgx.Tx, empresa, id int64) error {
	rows, err := tx.Query(ctx,
		`UPDATE stock_reservations s SET status='CANCELLED', updated_at=NOW()
		   FROM material_requisition_items l
		  WHERE l.requisition_id=$2 AND s.enterprise_id=$1 AND s.reference_type=$3
		    AND s.reference_item_code=l.id AND s.status='ACTIVE'
		  RETURNING s.item_code, s.mask, s.warehouse_id, s.quantity`, empresa, id, entity.ReferenceTypeRequisition)
	if err != nil {
		return fmt.Errorf("releasing requisition reservations: %w", err)
	}
	type reserva struct {
		item, almox int64
		mask        string
		qtd         decimal.Decimal
	}
	var lista []reserva
	for rows.Next() {
		var v reserva
		if err := rows.Scan(&v.item, &v.mask, &v.almox, &v.qtd); err != nil {
			rows.Close()
			return fmt.Errorf("releasing requisition reservations: %w", err)
		}
		lista = append(lista, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("releasing requisition reservations: %w", err)
	}
	for _, v := range lista {
		if err := adjustReservedTx(ctx, tx, empresa, v.item, v.mask, v.almox, v.qtd.Neg()); err != nil {
			return err
		}
	}
	return nil
}

// devolverEntregaDaRequisicaoTx tira da requisição o que foi estornado. A
// quantidade volta a ficar pendente, sem reserva; requisição encerrada à mão
// continua encerrada.
func devolverEntregaDaRequisicaoTx(ctx context.Context, tx pgx.Tx, empresa, codigo, item int64, mask string, qtd decimal.Decimal) error {
	req, err := scanRequisicao(tx.QueryRow(ctx, `SELECT `+requisicaoCols+` FROM material_requisitions WHERE enterprise_id=$1 AND code=$2 FOR UPDATE`, empresa, codigo))
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("requisição %d não encontrada", codigo))
	}
	if err != nil {
		return fmt.Errorf("locking material requisition: %w", err)
	}
	id := req.ID
	itens, err := tx.Query(ctx,
		`SELECT id, issued_qty FROM material_requisition_items
		  WHERE requisition_id=$1 AND item_code=$2 AND mask=$3 AND issued_qty > 0 ORDER BY sequence DESC FOR UPDATE`, id, item, mask)
	if err != nil {
		return fmt.Errorf("reading requisition lines: %w", err)
	}
	type linha struct {
		id  int64
		qtd decimal.Decimal
	}
	var linhas []linha
	for itens.Next() {
		var l linha
		if err := itens.Scan(&l.id, &l.qtd); err != nil {
			itens.Close()
			return fmt.Errorf("reading requisition lines: %w", err)
		}
		linhas = append(linhas, l)
	}
	itens.Close()
	if err := itens.Err(); err != nil {
		return fmt.Errorf("reading requisition lines: %w", err)
	}
	resta := qtd
	for _, l := range linhas {
		q := decimal.Min(l.qtd, resta)
		if !q.IsPositive() {
			break
		}
		if _, err := tx.Exec(ctx, `UPDATE material_requisition_items SET issued_qty = issued_qty - $2 WHERE id=$1`, l.id, q); err != nil {
			return fmt.Errorf("returning requisition line: %w", err)
		}
		resta = resta.Sub(q)
	}
	if req.Status == entity.RequisicaoCancelada {
		return nil
	}
	// O material voltou e a linha está pendente de novo: fica reservado para ela,
	// como na abertura.
	if req.Itens, err = lerItensDaRequisicao(ctx, tx, id); err != nil {
		return err
	}
	if err := reservarRequisicaoTx(ctx, tx, empresa, req); err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`UPDATE material_requisitions q SET updated_at=NOW(), status = CASE
		     WHEN EXISTS (SELECT 1 FROM material_requisition_items l WHERE l.requisition_id=q.id AND l.issued_qty > 0) THEN 'PARCIAL' ELSE 'ABERTA' END
		  WHERE q.id=$1`, id)
	if err != nil {
		return fmt.Errorf("reopening requisition: %w", err)
	}
	return nil
}

func rotuloRequisicao(s string) string {
	switch s {
	case entity.RequisicaoPendente:
		return "aguardando aprovação"
	case entity.RequisicaoReprovada:
		return "reprovada"
	case entity.RequisicaoAtendida:
		return "atendida"
	case entity.RequisicaoCancelada:
		return "cancelada"
	}
	return strings.ToLower(s)
}
