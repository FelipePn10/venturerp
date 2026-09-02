package bom_header

import (
	"context"
	"errors"
	"fmt"

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
const headerColumns = `id, item_code, mask, bom_type, version, status, valid_from, is_active, created_by, created_at, updated_at`

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
	if err := s.Scan(&h.ID, &h.ItemCode, &h.Mask, &h.BomType, &h.Version, &h.Status, &h.ValidFrom, &h.IsActive, &h.CreatedBy, &h.CreatedAt, &h.UpdatedAt); err != nil {
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

func (r *BomHeaderRepositorySQLC) UpdateStatus(ctx context.Context, id int64, status string) (*entity.BomHeader, error) {
	e, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `UPDATE bom_headers SET status=$3, updated_at=NOW()
		WHERE id=$1 AND enterprise_id=$2 RETURNING `+headerColumns, id, e, status)
	h, err := scanHeader(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("cabeçalho de estrutura %d não encontrado nesta empresa", id))
	}
	if err != nil {
		return nil, err
	}
	return h, nil
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
