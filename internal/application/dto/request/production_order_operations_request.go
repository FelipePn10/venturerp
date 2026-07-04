package request

type ExplodeRouteDTO struct {
	OrderID int64 `json:"order_id"`
	RouteID int64 `json:"route_id"`
}

type AdvanceOperationDTO struct {
	OperationID int64   `json:"operation_id"`
	Status      string  `json:"status"` // PENDING | IN_PROGRESS | DONE | SKIPPED
	ActualHours float64 `json:"actual_hours"`
	// ProducedQty (pieces) drives tool-life consumption for GOLPES/PECAS tools when
	// the operation is completed (DONE). HORAS tools consume ActualHours instead.
	ProducedQty float64 `json:"produced_qty"`
}
