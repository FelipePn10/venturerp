package bom_header

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/bom_header/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// O cabeçalho de estrutura é multiempresa desde a migração 000330. As consultas
// são escritas à mão para carregar o filtro por empresa em todas elas.
const headerColumns = `id, item_code, mask, bom_type, version, status, valid_from, is_active, created_by, created_at, updated_at, approved_by, approved_at, obsoleted_at, notes`

type BomHeaderRepositorySQLC struct {
	pool *pgxpool.Pool
}

// New recebe o pool porque toda consulta do cabeçalho é filtrada por empresa; o
// parâmetro sqlc.Queries é mantido para preservar a assinatura de composição.
func New(_ *sqlc.Queries, pool *pgxpool.Pool) domainrepo.BomHeaderRepository {
	return &BomHeaderRepositorySQLC{pool: pool}
}

type scanner interface {
	Scan(dest ...any) error
}

func scanHeader(s scanner) (*entity.BomHeader, error) {
	var h entity.BomHeader
	if err := s.Scan(&h.ID, &h.ItemCode, &h.Mask, &h.BomType, &h.Version, &h.Status, &h.ValidFrom, &h.IsActive, &h.CreatedBy, &h.CreatedAt, &h.UpdatedAt, &h.ApprovedBy, &h.ApprovedAt, &h.ObsoletedAt, &h.Notes); err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *BomHeaderRepositorySQLC) Create(ctx context.Context, h *entity.BomHeader) (*entity.BomHeader, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `INSERT INTO bom_headers (enterprise_id, item_code, mask, bom_type, version, status, valid_from, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING `+headerColumns,
		e, h.ItemCode, h.Mask, h.BomType, h.Version, h.Status, h.ValidFrom, h.CreatedBy)
	created, err := scanHeader(row)
	if err != nil {
		return nil, fmt.Errorf("criando cabeçalho de estrutura: %w", err)
	}
	return created, nil
}

func (r *BomHeaderRepositorySQLC) GetByID(ctx context.Context, id int64) (*entity.BomHeader, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `SELECT `+headerColumns+` FROM bom_headers WHERE id=$1 AND enterprise_id=$2`, id, e)
	h, err := scanHeader(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("cabeçalho de estrutura %d não encontrado nesta empresa", id))
	}
	if err != nil {
		return nil, err
	}
	return h, nil
}

func (r *BomHeaderRepositorySQLC) ListByItem(ctx context.Context, itemCode int64) ([]*entity.BomHeader, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+headerColumns+` FROM bom_headers
		WHERE item_code=$1 AND enterprise_id=$2 AND is_active = TRUE ORDER BY version DESC`, itemCode, e)
	if err != nil {
		return nil, fmt.Errorf("listando cabeçalhos de estrutura do item %d: %w", itemCode, err)
	}
	defer rows.Close()
	out := make([]*entity.BomHeader, 0)
	for rows.Next() {
		h, scanErr := scanHeader(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *BomHeaderRepositorySQLC) NextVersion(ctx context.Context, itemCode int64, mask string) (int32, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	var next int32
	err = r.pool.QueryRow(ctx, `SELECT (COALESCE(MAX(version),0)+1)::INTEGER FROM bom_headers
		WHERE item_code=$1 AND enterprise_id=$2 AND COALESCE(mask,'')=COALESCE($3,'')`, itemCode, e, mask).Scan(&next)
	if err != nil {
		return 0, err
	}
	return next, nil
}

// Linhas ativas da estrutura do item na máscara — o mesmo critério da explosão
// (linha sem máscara vale para todas), com todas as validades.
const linhasAtuais = `SELECT s.id, s.child_code, COALESCE(s.parent_mask,''), s.sequence, s.quantity, COALESCE(s.unit_of_measurement,''),
	COALESCE(s.loss_percentage,0), COALESCE(s.quantity_formula,''), s.start_date, s.end_date, COALESCE(s.is_coproduct,false)
	FROM item_structures s
	WHERE s.parent_code=$1 AND s.is_active AND (s.parent_mask IS NULL OR s.parent_mask = NULLIF($2,''))
	  AND EXISTS (SELECT 1 FROM items i WHERE i.code=s.parent_code AND i.enterprise_id=$3)
	ORDER BY s.sequence, s.id`

func scanLinhas(rows pgx.Rows) ([]entity.Line, []int64, error) {
	defer rows.Close()
	var out []entity.Line
	var ids []int64
	for rows.Next() {
		var l entity.Line
		var id int64
		if err := rows.Scan(&id, &l.ChildCode, &l.ParentMask, &l.Sequence, &l.Quantity, &l.Unit, &l.LossPercentage, &l.QuantityFormula, &l.StartDate, &l.EndDate, &l.IsCoproduct); err != nil {
			return nil, nil, err
		}
		out = append(out, l)
		ids = append(ids, id)
	}
	return out, ids, rows.Err()
}

func (r *BomHeaderRepositorySQLC) CurrentLines(ctx context.Context, itemCode int64, mask string) ([]entity.Line, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, linhasAtuais, itemCode, mask, e)
	if err != nil {
		return nil, fmt.Errorf("lendo a estrutura do item %d: %w", itemCode, err)
	}
	l, _, err := scanLinhas(rows)
	return l, err
}

func (r *BomHeaderRepositorySQLC) ApprovedLines(ctx context.Context, id int64) ([]entity.Line, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT COALESCE(structure_id,0), child_code, COALESCE(parent_mask,''), COALESCE(sequence,0), quantity, COALESCE(unit_of_measurement,''),
		COALESCE(loss_percentage,0), COALESCE(quantity_formula,''), start_date, end_date, is_coproduct
		FROM bom_header_lines WHERE bom_header_id=$1 AND enterprise_id=$2 ORDER BY sequence, id`, id, e)
	if err != nil {
		return nil, err
	}
	l, _, err := scanLinhas(rows)
	return l, err
}

func (r *BomHeaderRepositorySQLC) Approve(ctx context.Context, id int64, actor uuid.UUID, validFrom time.Time) (*entity.BomHeader, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	h, err := scanHeader(tx.QueryRow(ctx, `SELECT `+headerColumns+` FROM bom_headers WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, id, e))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("versão de estrutura %d não encontrada nesta empresa", id))
	}
	if err != nil {
		return nil, err
	}
	if err := h.PodeIrPara(entity.StatusApproved); err != nil {
		return nil, err
	}
	mask := ""
	if h.Mask != nil {
		mask = *h.Mask
	}
	rows, err := tx.Query(ctx, linhasAtuais, h.ItemCode, mask, e)
	if err != nil {
		return nil, err
	}
	linhas, ids, err := scanLinhas(rows)
	if err != nil {
		return nil, err
	}
	if len(linhas) == 0 {
		return nil, errorsuc.NewValidationError("a estrutura do item está vazia: cadastre os componentes (VENT0210) antes de aprovar a versão")
	}
	if _, err := tx.Exec(ctx, `UPDATE bom_headers SET status='OBSOLETE', obsoleted_at=NOW(), updated_at=NOW()
		WHERE enterprise_id=$1 AND item_code=$2 AND COALESCE(mask,'')=COALESCE($3,'') AND bom_type=$4
		  AND status='APPROVED' AND is_active AND id<>$5`, e, h.ItemCode, h.Mask, h.BomType, id); err != nil {
		return nil, err
	}
	aprovada, err := scanHeader(tx.QueryRow(ctx, `UPDATE bom_headers SET status='APPROVED', approved_by=$3, approved_at=NOW(),
		valid_from=COALESCE(valid_from,$4), updated_at=NOW() WHERE id=$1 AND enterprise_id=$2 RETURNING `+headerColumns, id, e, actor, validFrom))
	if err != nil {
		return nil, err
	}
	for i, l := range linhas {
		if _, err := tx.Exec(ctx, `INSERT INTO bom_header_lines(enterprise_id,bom_header_id,structure_id,child_code,parent_mask,sequence,quantity,unit_of_measurement,loss_percentage,quantity_formula,start_date,end_date,is_coproduct)
			VALUES($1,$2,$3,$4,NULLIF($5,''),$6,$7,NULLIF($8,''),$9,NULLIF($10,''),$11,$12,$13)`,
			e, id, ids[i], l.ChildCode, l.ParentMask, l.Sequence, l.Quantity, l.Unit, l.LossPercentage, l.QuantityFormula, l.StartDate, l.EndDate, l.IsCoproduct); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return aprovada, nil
}

func (r *BomHeaderRepositorySQLC) Obsolete(ctx context.Context, id int64) (*entity.BomHeader, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	h, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := h.PodeIrPara(entity.StatusObsolete); err != nil {
		return nil, err
	}
	return scanHeader(r.pool.QueryRow(ctx, `UPDATE bom_headers SET status='OBSOLETE', obsoleted_at=NOW(), updated_at=NOW()
		WHERE id=$1 AND enterprise_id=$2 AND status=$3 RETURNING `+headerColumns, id, e, h.Status))
}

// SituacaoParaProducao: o item está sob controle de versão quando tem versão de
// fabricação (MBOM) para a máscara (ou genérica); só produz com uma aprovada e
// vigente.
func (r *BomHeaderRepositorySQLC) SituacaoParaProducao(ctx context.Context, itemCode int64, mask string) (bool, bool, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return false, false, err
	}
	var controla, aprovada bool
	err = r.pool.QueryRow(ctx, `SELECT
		EXISTS(SELECT 1 FROM bom_headers WHERE enterprise_id=$1 AND item_code=$2 AND bom_type='MBOM' AND is_active AND (mask IS NULL OR mask = NULLIF($3,''))),
		EXISTS(SELECT 1 FROM bom_headers WHERE enterprise_id=$1 AND item_code=$2 AND bom_type='MBOM' AND is_active AND (mask IS NULL OR mask = NULLIF($3,''))
		       AND status='APPROVED' AND (valid_from IS NULL OR valid_from <= CURRENT_DATE))`, e, itemCode, mask).Scan(&controla, &aprovada)
	return controla, aprovada, err
}
