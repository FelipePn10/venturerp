package response

import (
	"time"

	"github.com/google/uuid"
)

// DeliveryPromiseParamsResponse is the API representation of delivery promise params.
type DeliveryPromiseParamsResponse struct {
	ID                      int64     `json:"id"`
	UseDeliveryPromise      bool      `json:"use_delivery_promise"`
	BlockedOrdersInPromise  bool      `json:"blocked_orders_in_promise"`
	DefaultOrderSort        string    `json:"default_order_sort"`
	ShowOrderValues         int       `json:"show_order_values"`
	BlockedExportInPromise  bool      `json:"blocked_export_in_promise"`
	BreakTankOccupation     bool      `json:"break_tank_occupation"`
	RecalculateAfterRelease bool      `json:"recalculate_after_release"`
	ReprogramLoadedOrders   bool      `json:"reprogram_loaded_orders"`
	AllowDeliveryDateChange bool      `json:"allow_delivery_date_change"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
	UpdatedBy               uuid.UUID `json:"updated_by"`
}
