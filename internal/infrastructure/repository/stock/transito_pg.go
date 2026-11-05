package stock

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

const transitoCols = `id, code, item_code, mask, lot, from_warehouse_id, transit_warehouse_id, to_warehouse_id,
	quantity, received_qty, returned_qty, lost_qty, status, expected_date, requisition_id, notes, created_at, closed_at`

func scanTransito(row pgx.Row) (*entity.Transito, error) {
	var t entity.Transito
	if err := row.Scan(&t.ID, &t.Code, &t.ItemCode, &t.Mask, &t.Lot, &t.DeAlmox, &t.AlmoxTransito, &t.ParaAlmox,
		&t.Quantidade, &t.Recebido, &t.Devolvido, &t.Perdido, &t.Status, &t.Previsao, &t.RequisicaoID, &t.Notas, &t.CreatedAt, &t.ClosedAt); err != nil {
		return nil, err
	}
	return &t, nil
}

// EnviarParaTransito tira o material da origem. Ele só entra no destino quando
// for recebido; até lá não conta para promessa em lugar nenhum.
func (r *StockRepositorySQLC) EnviarParaTransito(ctx context.Context, e entity.EnvioTransito) (*entity.Transito, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transit tx: %w", err)
	}
	defer tx.Rollback(ctx)
	t, err := enviarParaTransitoTx(ctx, tx, empresa, e)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transit: %w", err)
	}
	return t, nil
}

// almoxDeTransitoTx confere o almoxarifado informado ou acha o único da empresa.
func almoxDeTransitoTx(ctx context.Context, tx pgx.Tx, empresa, informado int64) (int64, error) {
	rows, err := tx.Query(ctx,
		`SELECT id FROM warehouse WHERE enterprise_id=$1 AND location::text='TRANSITO' AND ($2 = 0 OR id=$2) ORDER BY id`, empresa, informado)
	if err != nil {
		return 0, fmt.Errorf("reading transit warehouses: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("reading transit warehouses: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("reading transit warehouses: %w", err)
	}
	switch {
	case len(ids) == 1:
		return ids[0], nil
	case informado != 0:
		return 0, errorsuc.NewValidationError("o almoxarifado escolhido não é de trânsito")
	case len(ids) == 0:
		return 0, errorsuc.NewValidationError("cadastre um almoxarifado com localização Trânsito para usar a transferência em duas etapas")
	}
	return 0, errorsuc.NewValidationError("a empresa tem mais de um almoxarifado de trânsito: escolha qual usar")
}

func enviarParaTransitoTx(ctx context.Context, tx pgx.Tx, empresa int64, e entity.EnvioTransito) (*entity.Transito, error) {
	if e.DeAlmox == e.ParaAlmox {
		return nil, errorsuc.NewValidationError("origem e destino são o mesmo almoxarifado")
	}
	transito, err := almoxDeTransitoTx(ctx, tx, empresa, e.AlmoxTransito)
	if err != nil {
		return nil, err
	}
	if transito == e.DeAlmox || transito == e.ParaAlmox {
		return nil, errorsuc.NewValidationError("origem e destino não podem ser o almoxarifado de trânsito")
	}
	// A trava serializa a numeração: duas remessas ao mesmo tempo tirariam o mesmo código.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('stock_transit_code', $1))`, empresa); err != nil {
		return nil, fmt.Errorf("locking transit numbering: %w", err)
	}
	var codigo int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(code),0)+1 FROM stock_transit_shipments WHERE enterprise_id=$1`, empresa).Scan(&codigo); err != nil {
		return nil, fmt.Errorf("numbering transit: %w", err)
	}
	ref := entity.ReferenceTypeTransit
	if _, err := transferirTx(ctx, tx, empresa, entity.TransferenciaEstoque{
		ItemCode: e.ItemCode, Mask: e.Mask, DeAlmox: e.DeAlmox, ParaAlmox: transito, Quantidade: e.Quantidade,
		Lote: e.Lote, EnderecoDe: e.EnderecoDe, Notas: e.Notas, Ator: e.Ator, ReferenciaTipo: &ref, ReferenciaCodigo: &codigo,
	}, true); err != nil {
		return nil, err
	}
	t, err := scanTransito(tx.QueryRow(ctx,
		`INSERT INTO stock_transit_shipments (enterprise_id, code, item_code, mask, lot, from_warehouse_id, transit_warehouse_id,
		     to_warehouse_id, quantity, expected_date, requisition_id, notes, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING `+transitoCols,
		empresa, codigo, e.ItemCode, e.Mask, e.Lote, e.DeAlmox, transito, e.ParaAlmox, e.Quantidade, e.Previsao, e.RequisicaoID, e.Notas, e.Ator))
	if err != nil {
		return nil, fmt.Errorf("creating transit shipment: %w", err)
	}
	return t, nil
}

// BaixarTransito dá destino a parte do envio: chega, volta à origem ou se perde.
// Os movimentos ficam na referência do envio, e é por ele que se desfaz.
func (r *StockRepositorySQLC) BaixarTransito(ctx context.Context, b entity.BaixaTransito) (*entity.Transito, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transit tx: %w", err)
	}
	defer tx.Rollback(ctx)
	t, err := scanTransito(tx.QueryRow(ctx, `SELECT `+transitoCols+` FROM stock_transit_shipments WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, b.ID, empresa))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("envio em trânsito %d não encontrado", b.ID))
	}
	if err != nil {
		return nil, fmt.Errorf("locking transit shipment: %w", err)
	}
	if t.Status != entity.TransitoAberto {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o envio %d já foi encerrado", t.Code))
	}
	if b.Quantidade.GreaterThan(t.Pendente()) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o envio %d só tem %s em trânsito", t.Code, entity.QuantidadeBR(t.Pendente())))
	}
	ref := entity.ReferenceTypeTransit
	nota := fmt.Sprintf("Envio em trânsito %d", t.Code)
	if m := strings.TrimSpace(b.Motivo); m != "" {
		nota += ": " + m
	}
	coluna := ""
	switch b.Acao {
	case entity.TransitoReceber, entity.TransitoDevolver:
		destino := t.ParaAlmox
		coluna = "received_qty"
		if b.Acao == entity.TransitoDevolver {
			destino, coluna = t.DeAlmox, "returned_qty"
		}
		_, err = transferirTx(ctx, tx, empresa, entity.TransferenciaEstoque{
			ItemCode: t.ItemCode, Mask: t.Mask, DeAlmox: t.AlmoxTransito, ParaAlmox: destino, Quantidade: b.Quantidade,
			Lote: t.Lot, EnderecoPara: b.Endereco, Notas: &nota, Ator: b.Ator, ReferenciaTipo: &ref, ReferenciaCodigo: &t.Code,
		}, false)
	case entity.TransitoPerda:
		coluna = "lost_qty"
		err = CreateMovementTx(ctx, tx, empresa, &entity.StockMovement{
			ItemCode: t.ItemCode, Mask: t.Mask, WarehouseID: t.AlmoxTransito, MovementType: entity.MovementTypeOut,
			Quantity: b.Quantidade, Lot: t.Lot, ReferenceType: &ref, ReferenceCode: &t.Code, Notes: &nota, CreatedBy: b.Ator,
		})
	default:
		return nil, errorsuc.NewValidationError("escolha entre receber, devolver ou lançar perda")
	}
	if err != nil {
		return nil, err
	}
	out, err := scanTransito(tx.QueryRow(ctx,
		`UPDATE stock_transit_shipments SET `+coluna+` = `+coluna+` + $2,
		     status = CASE WHEN quantity - received_qty - returned_qty - lost_qty - $2 <= 0 THEN 'ENCERRADA' ELSE status END,
		     closed_at = CASE WHEN quantity - received_qty - returned_qty - lost_qty - $2 <= 0 THEN NOW() END
		  WHERE id=$1 RETURNING `+transitoCols, t.ID, b.Quantidade))
	if err != nil {
		return nil, fmt.Errorf("updating transit shipment: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transit: %w", err)
	}
	return out, nil
}

// ListarTransitos lista os envios da empresa, os mais recentes primeiro.
func (r *StockRepositorySQLC) ListarTransitos(ctx context.Context, soAbertos bool) ([]entity.Transito, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+transitoCols+` FROM stock_transit_shipments
		  WHERE enterprise_id=$1 AND (NOT $2 OR status='EM_TRANSITO') ORDER BY code DESC LIMIT 500`, empresa, soAbertos)
	if err != nil {
		return nil, fmt.Errorf("listing transit shipments: %w", err)
	}
	defer rows.Close()
	out := []entity.Transito{}
	for rows.Next() {
		t, err := scanTransito(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning transit shipment: %w", err)
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}
