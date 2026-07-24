package sales_quotation

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	salesorderrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/sales_order"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConversionUnitOfWork struct{ pool *pgxpool.Pool }

func NewConversionUnitOfWork(pool *pgxpool.Pool) *ConversionUnitOfWork {
	return &ConversionUnitOfWork{pool: pool}
}
func (u *ConversionUnitOfWork) Execute(ctx context.Context, quotationCode int64, fn func(orderrepo.SalesOrderRepository) (*orderentity.SalesOrder, error)) (*orderentity.SalesOrder, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	orders := salesorderrepo.NewSalesOrderRepositorySQLC(sqlc.New(tx))
	created, err := fn(orders)
	if err != nil {
		return nil, err
	}
	tag, err := tx.Exec(ctx, `UPDATE public.sales_quotations SET status='ATTENDED',attended_reason='Convertido em pedido de venda',attended_at=NOW(),converted_sales_order_code=$3,converted_at=NOW(),updated_at=NOW() WHERE code=$1 AND enterprise_code=$2 AND converted_sales_order_code IS NULL AND status NOT IN ('CANCELLED','ATTENDED','EXPIRED')`, quotationCode, tenantID, created.Code)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf("sales quotation %d not found or cannot be converted", quotationCode)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO public.sales_quotation_events(sales_quotation_code,event_type,reason) VALUES($1,'CONVERT','Convertido em pedido de venda')`, quotationCode); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

var _ ports.SalesQuotationConversionUnitOfWork = (*ConversionUnitOfWork)(nil)
