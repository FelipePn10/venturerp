package item_conversion

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"

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
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateItemUnitConversion(ctx, sqlc.CreateItemUnitConversionParams{
		EnterpriseID:    empresa,
		ItemCode:        c.ItemCode,
		Mask:            c.Mask,
		FromUom:         c.FromUOM,
		ToUom:           c.ToUOM,
		Factor:          pgutil.ToPgNumericFromFloat64(c.Factor),
		RoundingPercent: pgutil.ToPgNumericFromFloat64(c.RoundingPercent), ToleranceValue: pgutil.ToPgNumericFromFloat64(c.ToleranceValue), ToleranceType: c.ToleranceType,
		CreatedBy: pgutil.ToPgUUID(c.CreatedBy),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("item %d não encontrado", c.ItemCode))
	}
	if err != nil {
		return nil, fmt.Errorf("creating item unit conversion: %w", err)
	}
	return conversionToEntity(row), nil
}

func (r *ItemConversionRepositorySQLC) ListByItem(ctx context.Context, itemCode int64) ([]*entity.ItemUnitConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListItemUnitConversions(ctx, sqlc.ListItemUnitConversionsParams{ItemCode: itemCode, EnterpriseID: empresa})
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
	return r.GetConfigured(ctx, itemCode, "", fromUOM, toUOM)
}

func (r *ItemConversionRepositorySQLC) GetConfigured(ctx context.Context, itemCode int64, mask, fromUOM, toUOM string) (*entity.ItemUnitConversion, error) {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetItemUnitConversion(ctx, sqlc.GetItemUnitConversionParams{
		EnterpriseID: empresa,
		ItemCode:     itemCode,
		Mask:         mask,
		FromUom:      fromUOM,
		ToUom:        toUOM,
	})
	if err != nil {
		return nil, err
	}
	return conversionToEntity(row), nil
}

func (r *ItemConversionRepositorySQLC) AcceptsFractional(ctx context.Context, itemCode int64) (bool, error) {
	var value bool
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	err = r.pool.QueryRow(ctx, `SELECT accepts_fractional_quantity FROM items WHERE code=$1 AND enterprise_id=$2`, itemCode, empresa).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, errorsuc.NewNotFoundError(fmt.Sprintf("item %d não encontrado", itemCode))
	}
	return value, err
}

func (r *ItemConversionRepositorySQLC) Delete(ctx context.Context, id int64) error {
	empresa, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	n, err := r.q.DeleteItemUnitConversion(ctx, sqlc.DeleteItemUnitConversionParams{ID: id, EnterpriseID: empresa})
	if err != nil {
		return err
	}
	if n == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("conversão %d não encontrada", id))
	}
	return nil
}

func conversionToEntity(row sqlc.ItemUnitConversion) *entity.ItemUnitConversion {
	return &entity.ItemUnitConversion{
		ID:              row.ID,
		ItemCode:        row.ItemCode,
		Mask:            row.Mask,
		FromUOM:         row.FromUom,
		ToUOM:           row.ToUom,
		Factor:          pgutil.FromPgNumericToFloat64(row.Factor),
		RoundingPercent: pgutil.FromPgNumericToFloat64(row.RoundingPercent), ToleranceValue: pgutil.FromPgNumericToFloat64(row.ToleranceValue), ToleranceType: row.ToleranceType,
		IsActive:  row.IsActive,
		CreatedAt: pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy: pgutil.FromPgUUID(row.CreatedBy),
	}
}
