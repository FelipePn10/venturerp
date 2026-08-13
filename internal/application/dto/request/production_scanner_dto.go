package request

type CreateProductionScanTokenDTO struct {
	ProductionOrderID int64   `json:"production_order_id"`
	OperationID       *int64  `json:"operation_id,omitempty"`
	ValidUntil        *string `json:"valid_until,omitempty"`
}

type ProductionScanDTO struct {
	Token             string  `json:"token"`
	Action            string  `json:"action"`
	IdempotencyKey    string  `json:"idempotency_key"`
	DeviceID          string  `json:"device_id"`
	EmployeeID        *int64  `json:"employee_id,omitempty"`
	GoodQuantity      string  `json:"good_quantity,omitempty"`
	ScrapQuantity     string  `json:"scrap_quantity,omitempty"`
	Hours             string  `json:"hours,omitempty"`
	ScrapReason       *string `json:"scrap_reason,omitempty"`
	CompleteOperation bool    `json:"complete_operation,omitempty"`
}
