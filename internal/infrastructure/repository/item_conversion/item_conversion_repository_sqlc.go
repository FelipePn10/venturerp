package item_conversion

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/item_conversion/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemConversionRepositorySQLC struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func New(q *sqlc.Queries, pool *pgxpool.Pool) domainrepo.ItemConversionRepository {
	return &ItemConversionRepositorySQLC{q: q, pool: pool}
}

func (r *ItemConversionRepositorySQLC) Create(ctx context.Context, c *entity.ItemUnitConversion) (*entity.ItemUnitConversion, error) {
	row, err := r.q.CreateItemUnitConversion(ctx, sqlc.CreateItemUnitConversionParams{
		ItemCode:  c.ItemCode,
		FromUom:   c.FromUOM,
		ToUom:     c.ToUOM,
		Factor:    pgutil.ToPgNumericFromFloat64(c.Factor),
		CreatedBy: pgutil.ToPgUUID(c.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating item unit conversion: %w", err)
	}
	return conversionToEntity(row), nil
}

func (r *ItemConversionRepositorySQLC) ListByItem(ctx context.Context, itemCode int64) ([]*entity.ItemUnitConversion, error) {
	rows, err := r.q.ListItemUnitConversions(ctx, itemCode)
	if err != nil {
		return nil, err
	}
	out := make([]*entity.ItemUnitConversion, 0, len(rows))
	for _, row := range rows {
		out = append(out, conversionToEntity(row))
	}
	return out, nil
}

func (r *ItemConversionRepositorySQLC) Get(ctx context.Context, itemCode int64, fromUOM, toUOM string) (*entity.ItemUnitConversion, error) {
	row, err := r.q.GetItemUnitConversion(ctx, sqlc.GetItemUnitConversionParams{
		ItemCode: itemCode,
		FromUom:  fromUOM,
		ToUom:    toUOM,
	})
	if err != nil {
		return nil, err
	}
	return conversionToEntity(row), nil
}

func (r *ItemConversionRepositorySQLC) Delete(ctx context.Context, id int64) error {
	return r.q.DeleteItemUnitConversion(ctx, id)
}

func conversionToEntity(row sqlc.ItemUnitConversion) *entity.ItemUnitConversion {
	return &entity.ItemUnitConversion{
		ID:        row.ID,
		ItemCode:  row.ItemCode,
		FromUOM:   row.FromUom,
		ToUOM:     row.ToUom,
		Factor:    pgutil.FromPgNumericToFloat64(row.Factor),
		IsActive:  row.IsActive,
		CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy: pgutil.FromPgUUID(row.CreatedBy),
	}
}
