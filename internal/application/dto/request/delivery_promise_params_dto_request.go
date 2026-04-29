package request

import "github.com/google/uuid"

type UpdateDeliveryPromiseParamsDTO struct {
	UseDeliveryPromise      bool      `json:"use_delivery_promise"`
	BlockedOrdersInPromise  bool      `json:"blocked_orders_in_promise"`
	DefaultOrderSort        string    `json:"default_order_sort"`
	ShowOrderValues         int       `json:"show_order_values"`
	BlockedExportInPromise  bool      `json:"blocked_export_in_promise"`
	BreakTankOccupation     bool      `json:"break_tank_occupation"`
	RecalculateAfterRelease bool      `json:"recalculate_after_release"`
	ReprogramLoadedOrders   bool      `json:"reprogram_loaded_orders"`
	AllowDeliveryDateChange bool      `json:"allow_delivery_date_change"`
	UpdatedBy               uuid.UUID `json:"updated_by"`
}
