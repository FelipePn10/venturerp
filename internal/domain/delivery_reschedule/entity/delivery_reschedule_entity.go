package entity

import (
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
)

type DeliveryReschedule struct {
	Code           int64
	SalesOrderCode int64
	ItemCode       valueobject.ItemCode
	OldDate        time.Time
	NewDate        time.Time
	Reason         *string
	CreatedAt      time.Time
	CreatedBy      uuid.UUID
}
