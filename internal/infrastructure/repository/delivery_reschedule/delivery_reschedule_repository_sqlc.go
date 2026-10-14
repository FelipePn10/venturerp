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

// Reprogramação é da empresa do pedido de venda: a coluna é enterprise_code e
// o filtro usa tenant.Code (o ID só coincide com o código na empresa 1).

func (r *DeliveryRescheduleRepositorySQLC) Create(
	ctx context.Context,
	res *entity.DeliveryReschedule,
) (*entity.DeliveryReschedule, error) {
	empresa, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.CreateDeliveryReschedule(
		ctx,
		sqlc.CreateDeliveryRescheduleParams{
			SalesOrderCode: res.SalesOrderCode,
			ItemCode:       int64(res.ItemCode),
			OldDate:        pgutil.ToPgDate(res.OldDate),
			NewDate:        pgutil.ToPgDate(res.NewDate),
			Reason:         pgutil.ToPgTextFromPtr(res.Reason),
			CreatedBy:      pgutil.ToPgUUID(res.CreatedBy),
			EnterpriseCode: empresa,
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("pedido de venda %d não encontrado", res.SalesOrderCode))
		}
		return nil, fmt.Errorf("creating delivery reschedule: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *DeliveryRescheduleRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.DeliveryReschedule, error) {
	empresa, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	row, err := r.q.GetDeliveryRescheduleByCode(ctx, sqlc.GetDeliveryRescheduleByCodeParams{Code: code, EnterpriseCode: empresa})
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
	empresa, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListReschedulesByOrder(ctx, sqlc.ListReschedulesByOrderParams{SalesOrderCode: salesOrderCode, EnterpriseCode: empresa})
	if err != nil {
		return nil, fmt.Errorf("listing reschedules by order: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *DeliveryRescheduleRepositorySQLC) ListByItem(
	ctx context.Context,
	itemCode valueobject.ItemCode,
) ([]*entity.DeliveryReschedule, error) {
	empresa, err := tenant.Code(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListReschedulesByItem(ctx, sqlc.ListReschedulesByItemParams{ItemCode: int64(itemCode), EnterpriseCode: empresa})
	if err != nil {
		return nil, fmt.Errorf("listing reschedules by item: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *DeliveryRescheduleRepositorySQLC) Delete(
	ctx context.Context,
	id int64,
) error {
	empresa, err := tenant.Code(ctx)
	if err != nil {
		return err
	}
	n, err := r.q.DeleteDeliveryReschedule(ctx, sqlc.DeleteDeliveryRescheduleParams{Code: id, EnterpriseCode: empresa})
	if err != nil {
		return fmt.Errorf("deleting delivery reschedule: %w", err)
	}
	if n == 0 {
		return errorsuc.NewNotFoundError(fmt.Sprintf("reprogramação de entrega %d não encontrada", id))
	}
	return nil
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
