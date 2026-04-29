package request

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
)

type CreateDeliveryRescheduleDTO struct {
	Code           int64                `json:"code"`
	SalesOrderCode int64                `json:"sales_order_code"`
	ItemCode       valueobject.ItemCode `json:"item_code"`
	OldDate        time.Time            `json:"old_date"`
	NewDate        time.Time            `json:"new_date"`
	Reason         *string              `json:"reason,omitempty"`
	CreatedBy      uuid.UUID            `json:"created_by"`
}
