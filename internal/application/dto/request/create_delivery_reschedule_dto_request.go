package request

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
)

type CreateDeliveryRescheduleDTO struct {
	SalesOrderCode int64                `json:"sales_order_code"`
	ItemCode       valueobject.ItemCode `json:"item_code"`
	OldDate        time.Time            `json:"old_date"`
	NewDate        time.Time            `json:"new_date"`
	Reason         *string              `json:"reason,omitempty"`
}

type DeliveryRescheduleBatchItemDTO struct {
	SalesOrderItemCode int64                `json:"sales_order_item_code"`
	ItemCode           valueobject.ItemCode `json:"item_code"`
	OldDate            time.Time            `json:"old_date"`
	NewDate            time.Time            `json:"new_date"`
	Reason             *string              `json:"reason,omitempty"`
}

type IntegratedDeliveryRescheduleBatchDTO struct {
	SalesOrderCode int64                            `json:"sales_order_code"`
	IdempotencyKey string                           `json:"idempotency_key"`
	Items          []DeliveryRescheduleBatchItemDTO `json:"items"`
}
