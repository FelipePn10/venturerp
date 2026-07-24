package ports

import (
	"context"

	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	orderrepo "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
)

type SalesQuotationConversionUnitOfWork interface {
	Execute(ctx context.Context, quotationCode int64, fn func(orderrepo.SalesOrderRepository) (*orderentity.SalesOrder,error)) (*orderentity.SalesOrder,error)
}
