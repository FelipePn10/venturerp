package response

import (
	"time"

	"github.com/google/uuid"
)

// DeliveryRescheduleResponse is the API representation of a delivery reschedule.
type DeliveryRescheduleResponse struct {
	Code           int64     `json:"code"`
	SalesOrderCode int64     `json:"sales_order_code"`
	ItemCode       int64     `json:"item_code"`
	OldDate        time.Time `json:"old_date"`
	NewDate        time.Time `json:"new_date"`
	Reason         *string   `json:"reason,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      uuid.UUID `json:"created_by"`
}
