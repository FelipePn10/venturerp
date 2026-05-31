package item_supplier

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/item_supplier/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemSupplierRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool *pgxpool.Pool) domainrepo.ItemSupplierRepository {
	return &ItemSupplierRepositorySQLC{q: q, pool: pool}
}

func (r *ItemSupplierRepositorySQLC) Upsert(ctx context.Context, s *entity.ItemPreferredSupplier) (*entity.ItemPreferredSupplier, error) {
	row, err := r.q.UpsertItemPreferredSupplier(ctx, sqlc.UpsertItemPreferredSupplierParams{
		ItemCode:            s.ItemCode,
		SupplierCode:        s.SupplierCode,
		Ranking:             s.Ranking,
		SupplierItemCode:    pgutil.ToPgTextFromPtr(s.SupplierItemCode),
		SupplierDescription: pgutil.ToPgTextFromPtr(s.SupplierDescription),
		Uom:                 pgutil.ToPgTextFromPtr(s.UOM),
		LeadTimeDays:        s.LeadTimeDays,
		CreatedBy:           pgutil.ToPgUUID(s.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("upserting item preferred supplier: %w", err)
	}
	return toEntity(row), nil
}

func (r *ItemSupplierRepositorySQLC) ListByItem(ctx context.Context, itemCode int64) ([]*entity.ItemPreferredSupplier, error) {
	rows, err := r.q.ListItemPreferredSuppliers(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.ItemPreferredSupplier, 0, len(rows))
	for _, row := range rows {
		out = append(out, toEntity(row))
	}
	return out, nil
}

func (r *ItemSupplierRepositorySQLC) GetPreferred(ctx context.Context, itemCode int64) (*entity.ItemPreferredSupplier, error) {
	row, err := r.q.GetPreferredSupplierForItem(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	return toEntity(row), nil
}

func (r *ItemSupplierRepositorySQLC) Delete(ctx context.Context, id int64) error {
	return r.q.DeleteItemPreferredSupplier(ctx, id)
}

func toEntity(row sqlc.ItemPreferredSupplier) *entity.ItemPreferredSupplier {
	return &entity.ItemPreferredSupplier{
		ID:                  row.ID,
		ItemCode:            row.ItemCode,
		SupplierCode:        row.SupplierCode,
		Ranking:             row.Ranking,
		SupplierItemCode:    pgutil.FromPgTextPtr(row.SupplierItemCode),
		SupplierDescription: pgutil.FromPgTextPtr(row.SupplierDescription),
		UOM:                 pgutil.FromPgTextPtr(row.Uom),
		LeadTimeDays:        row.LeadTimeDays,
		IsActive:            row.IsActive,
		CreatedAt:           pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:           pgutil.FromPgUUID(row.CreatedBy),
	}
}
