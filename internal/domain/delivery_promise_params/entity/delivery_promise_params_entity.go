package entity

import (
	"time"

	"github.com/google/uuid"
)

type DeliveryPromiseParams struct {
	ID                      int64
	UseDeliveryPromise      bool
	BlockedOrdersInPromise  bool
	DefaultOrderSort        string
	ShowOrderValues         int
	BlockedExportInPromise  bool
	BreakTankOccupation     bool
	RecalculateAfterRelease bool
	ReprogramLoadedOrders   bool
	AllowDeliveryDateChange bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
	UpdatedBy               uuid.UUID
}
