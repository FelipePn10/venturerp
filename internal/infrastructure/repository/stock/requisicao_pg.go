package stock

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

const requisicaoCols = `id, code, warehouse_id, requester_name, requester_employee_code, cost_center_code, needed_date,
	status, notes, created_by, created_at, updated_at`

func scanRequisicao(row pgx.Row) (*entity.RequisicaoMaterial, error) {
	var r entity.RequisicaoMaterial
	if err := row.Scan(&r.ID, &r.Code, &r.WarehouseID, &r.RequesterName, &r.RequesterEmployeeCode, &r.CostCenterCode,
		&r.NeededDate, &r.Status, &r.Notes, &r.CreatedBy, &r.CreatedAt, &r.UpdatedAt); err != nil {
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
	var ok bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM warehouse WHERE id=$1 AND enterprise_id=$2)`, req.WarehouseID, empresa).Scan(&ok); err != nil {
		return nil, fmt.Errorf("checking requisition warehouse: %w", err)
	}
	if !ok {
		return nil, errorsuc.NewValidationError("almoxarifado da requisição não existe nesta empresa")
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
	if req.Code <= 0 {
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(code),0)+1 FROM material_requisitions WHERE enterprise_id=$1`, empresa).Scan(&req.Code); err != nil {
			return nil, fmt.Errorf("next requisition code: %w", err)
		}
	}
	criada, err := scanRequisicao(tx.QueryRow(ctx,
		`INSERT INTO material_requisitions (enterprise_id, code, warehouse_id, requester_name, requester_employee_code,
		     cost_center_code, needed_date, status, notes, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'ABERTA',$8,$9) RETURNING `+requisicaoCols,
		empresa, req.Code, req.WarehouseID, req.RequesterName, req.RequesterEmployeeCode, req.CostCenterCode,
		req.NeededDate, req.Notes, req.CreatedBy))
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
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing requisition: %w", err)
	}
	return criada, nil
}

func (r *StockRepositorySQLC) itensDaRequisicao(ctx context.Context, q interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, id int64) ([]entity.RequisicaoMaterialItem, error) {
	rows, err := q.Query(ctx,
		`SELECT id, sequence, item_code, mask, quantity, issued_qty, notes FROM material_requisition_items
		  WHERE requisition_id=$1 ORDER BY sequence`, id)
	if err != nil {
		return nil, fmt.Errorf("listing requisition lines: %w", err)
	}
	defer rows.Close()
	var out []entity.RequisicaoMaterialItem
	for rows.Next() {
		var it entity.RequisicaoMaterialItem
		if err := rows.Scan(&it.ID, &it.Sequence, &it.ItemCode, &it.Mask, &it.Quantity, &it.IssuedQty, &it.Notes); err != nil {
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
	if req.Status == entity.RequisicaoAtendida || req.Status == entity.RequisicaoCancelada {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a requisição %d está %s: não há o que atender", req.Code, req.Status))
	}
	itens, err := r.itensDaRequisicao(ctx, tx, id)
	if err != nil {
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
			return nil, errorsuc.NewValidationError(fmt.Sprintf("a linha %d não é desta requisição", l.ItemID))
		}
		if !l.Quantity.IsPositive() {
			continue
		}
		if l.Quantity.GreaterThan(it.Pendente()) {
			return nil, errorsuc.NewValidationError(fmt.Sprintf("linha %d: entrega de %s passa do pendente (%s)", it.Sequence, l.Quantity, it.Pendente()))
		}
		mov := &entity.StockMovement{
			ItemCode: it.ItemCode, Mask: it.Mask, WarehouseID: req.WarehouseID, MovementType: entity.MovementTypeOut,
			Quantity: l.Quantity, Lot: l.Lot, Address: l.Address,
			ReferenceType: &ref, ReferenceCode: &codigo, Notes: &nota, CreatedBy: ator, CostCenterCode: req.CostCenterCode,
		}
		if err := CreateMovementTx(ctx, tx, empresa, mov); err != nil {
			return nil, fmt.Errorf("linha %d: %w", it.Sequence, err)
		}
		if _, err := tx.Exec(ctx, `UPDATE material_requisition_items SET issued_qty = issued_qty + $2 WHERE id=$1`, it.ID, l.Quantity); err != nil {
			return nil, fmt.Errorf("updating issued quantity: %w", err)
		}
		it.IssuedQty = it.IssuedQty.Add(l.Quantity)
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

// CancelarRequisicaoMaterial encerra o que falta atender. O já entregue fica.
func (r *StockRepositorySQLC) CancelarRequisicaoMaterial(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var status string
	err = r.pool.QueryRow(ctx, `SELECT status FROM material_requisitions WHERE id=$1 AND enterprise_id=$2`, id, empresa).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return errorsuc.NewNotFoundError(fmt.Sprintf("requisição %d não encontrada", id))
	}
	if err != nil {
		return fmt.Errorf("reading requisition status: %w", err)
	}
	if status == entity.RequisicaoAtendida || status == entity.RequisicaoCancelada {
		return errorsuc.NewValidationError(fmt.Sprintf("a requisição está %s: não há o que cancelar", status))
	}
	_, err = r.pool.Exec(ctx, `UPDATE material_requisitions SET status='CANCELADA', updated_at=NOW() WHERE id=$1 AND enterprise_id=$2`, id, empresa)
	if err != nil {
		return fmt.Errorf("cancelling requisition: %w", err)
	}
	return nil
}
