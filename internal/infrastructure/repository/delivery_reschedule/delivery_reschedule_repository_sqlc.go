package delivery_reschedule

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"

	"github.com/jackc/pgx/v5"

	"github.com/FelipePn10/panossoerp/internal/domain/delivery_reschedule/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

func (r *DeliveryRescheduleRepositorySQLC) Create(
	ctx context.Context,
	res *entity.DeliveryReschedule,
) (*entity.DeliveryReschedule, error) {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return nil, err
		}
		row := r.pool.QueryRow(ctx, `INSERT INTO delivery_reschedules(sales_order_code,item_code,old_date,new_date,reason,created_by,enterprise_code) SELECT $1,$2,$3,$4,$5,$6,$7 WHERE EXISTS(SELECT 1 FROM sales_orders WHERE code=$1 AND enterprise_code=$7) RETURNING code,created_at`, res.SalesOrderCode, int64(res.ItemCode), res.OldDate, res.NewDate, res.Reason, res.CreatedBy, enterpriseID)
		if err := row.Scan(&res.Code, &res.CreatedAt); err != nil {
			return nil, fmt.Errorf("criando reprogramação na empresa autenticada: %w", err)
		}
		return res, nil
	}

	row, err := r.q.CreateDeliveryReschedule(
		ctx,
		sqlc.CreateDeliveryRescheduleParams{
			SalesOrderCode: res.SalesOrderCode,
			ItemCode:       int64(res.ItemCode),
			OldDate:        pgutil.ToPgDate(res.OldDate),
			NewDate:        pgutil.ToPgDate(res.NewDate),

			Reason: pgutil.ToPgTextFromPtr(res.Reason),

			CreatedBy: pgutil.ToPgUUID(res.CreatedBy),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("creating delivery reschedule: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *DeliveryRescheduleRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.DeliveryReschedule, error) {

	row, err := r.q.GetDeliveryRescheduleByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("reprogramação de entrega %d não encontrada", code))
		}
		return nil, fmt.Errorf("fetching delivery reschedule: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *DeliveryRescheduleRepositorySQLC) ListByOrder(
	ctx context.Context,
	salesOrderCode int64,
) ([]*entity.DeliveryReschedule, error) {
	if r.pool != nil {
		enterpriseID, err := tenant.ID(ctx)
		if err != nil {
			return nil, err
		}
		rows, err := r.pool.Query(ctx, `SELECT code,sales_order_code,item_code,old_date,new_date,reason,created_at,created_by FROM delivery_reschedules WHERE sales_order_code=$1 AND enterprise_code=$2 ORDER BY created_at DESC`, salesOrderCode, enterpriseID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []*entity.DeliveryReschedule
		for rows.Next() {
			v := new(entity.DeliveryReschedule)
			var item int64
			if err = rows.Scan(&v.Code, &v.SalesOrderCode, &item, &v.OldDate, &v.NewDate, &v.Reason, &v.CreatedAt, &v.CreatedBy); err != nil {
				return nil, err
			}
			v.ItemCode = valueobject.ItemCode(item)
			out = append(out, v)
		}
		return out, rows.Err()
	}

	rows, err := r.q.ListReschedulesByOrder(ctx, salesOrderCode)
	if err != nil {
		return nil, fmt.Errorf("listing reschedules by order: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *DeliveryRescheduleRepositorySQLC) ListByItem(
	ctx context.Context,
	itemCode valueobject.ItemCode,
) ([]*entity.DeliveryReschedule, error) {

	rows, err := r.q.ListReschedulesByItem(ctx, int64(itemCode))
	if err != nil {
		return nil, fmt.Errorf("listing reschedules by item: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *DeliveryRescheduleRepositorySQLC) Delete(
	ctx context.Context,
	id int64,
) error {
	return r.q.DeleteDeliveryReschedule(ctx, id)
}

func rowToEntity(row sqlc.DeliveryReschedule) *entity.DeliveryReschedule {
	e := &entity.DeliveryReschedule{
		Code:           row.Code,
		SalesOrderCode: row.SalesOrderCode,
		ItemCode:       valueobject.ItemCode(row.ItemCode),
		OldDate:        pgutil.FromPgDate(row.OldDate),
		NewDate:        pgutil.FromPgDate(row.NewDate),
		CreatedAt:      pgutil.FromPgTimestamptz(row.CreatedAt),
		CreatedBy:      pgutil.FromPgUUID(row.CreatedBy),
	}

	e.Reason = pgutil.FromPgTextPtr(row.Reason)

	return e
}

func rowsToEntities(rows []sqlc.DeliveryReschedule) []*entity.DeliveryReschedule {
	out := make([]*entity.DeliveryReschedule, 0, len(rows))

	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}

	return out
}
