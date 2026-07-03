package bom_header

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/bom_header/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/bom_header/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type BomHeaderRepositorySQLC struct {
	q *sqlc.Queries
}

func New(q *sqlc.Queries) domainrepo.BomHeaderRepository {
	return &BomHeaderRepositorySQLC{q: q}
}

func (r *BomHeaderRepositorySQLC) Create(ctx context.Context, h *entity.BomHeader) (*entity.BomHeader, error) {
	row, err := r.q.CreateBomHeader(ctx, sqlc.CreateBomHeaderParams{
		ItemCode:  h.ItemCode,
		Mask:      pgutil.ToPgTextFromPtr(h.Mask),
		BomType:   h.BomType,
		Version:   h.Version,
		Status:    h.Status,
		ValidFrom: pgutil.ToPgDateFromPtr(h.ValidFrom),
		CreatedBy: pgutil.ToPgUUID(h.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating bom header: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *BomHeaderRepositorySQLC) GetByID(ctx context.Context, id int64) (*entity.BomHeader, error) {
	row, err := r.q.GetBomHeader(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching bom header %d: %w", id, err)
	}
	return rowToEntity(row), nil
}

func (r *BomHeaderRepositorySQLC) ListByItem(ctx context.Context, itemCode int64) ([]*entity.BomHeader, error) {
	rows, err := r.q.ListBomHeadersByItem(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("listing bom headers for item %d: %w", itemCode, err)
	}
	out := make([]*entity.BomHeader, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out, nil
}

func (r *BomHeaderRepositorySQLC) UpdateStatus(ctx context.Context, id int64, status string) (*entity.BomHeader, error) {
	row, err := r.q.UpdateBomHeaderStatus(ctx, sqlc.UpdateBomHeaderStatusParams{ID: id, Status: status})
	if err != nil {
		return nil, fmt.Errorf("updating bom header status: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *BomHeaderRepositorySQLC) NextVersion(ctx context.Context, itemCode int64, mask string) (int32, error) {
	return r.q.NextBomVersion(ctx, sqlc.NextBomVersionParams{ItemCode: itemCode, Mask: pgutil.ToPgText(mask)})
}

func rowToEntity(row sqlc.BomHeader) *entity.BomHeader {
	e := &entity.BomHeader{
		ID:        row.ID,
		ItemCode:  row.ItemCode,
		BomType:   row.BomType,
		Version:   row.Version,
		Status:    row.Status,
		ValidFrom: pgutil.FromPgDateToPtr(row.ValidFrom),
		IsActive:  row.IsActive,
		CreatedBy: pgutil.FromPgUUID(row.CreatedBy),
		CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt: pgutil.FromPgTimestamptz(row.UpdatedAt),
	}
	if row.Mask.Valid {
		v := row.Mask.String
		e.Mask = &v
	}
	return e
}
