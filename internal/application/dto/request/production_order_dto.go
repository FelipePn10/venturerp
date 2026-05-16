package request

import "github.com/google/uuid"

type CreateProductionOrderDTO struct {
	PlannedOrderID *int64    `json:"planned_order_id,omitempty"`
	ItemCode       int64     `json:"item_code"`
	Mask           string    `json:"mask"`
	PlannedQty     float64   `json:"planned_qty"`
	StartDate      *string   `json:"start_date,omitempty"`
	EndDate        *string   `json:"end_date,omitempty"`
	MachineID      *int64    `json:"machine_id,omitempty"`
	CostCenterID   *int64    `json:"cost_center_id,omitempty"`
	EmployeeID     *int64    `json:"employee_id,omitempty"`
	Priority       *string   `json:"priority,omitempty"`
	Notes          *string   `json:"notes,omitempty"`
	CreatedBy      uuid.UUID `json:"created_by"`
}

type StartProductionOrderDTO struct {
	ID        int64  `json:"id"`
	StartDate string `json:"start_date"`
}

type AddAppointmentDTO struct {
	ProductionOrderID int64     `json:"production_order_id"`
	MachineID         *int64    `json:"machine_id,omitempty"`
	EmployeeID        *int64    `json:"employee_id,omitempty"`
	AppointmentDate   string    `json:"appointment_date"`
	StartTime         *string   `json:"start_time,omitempty"`
	EndTime           *string   `json:"end_time,omitempty"`
	ProducedQty       float64   `json:"produced_qty"`
	ScrappedQty       float64   `json:"scrapped_qty"`
	ScrapReason       *string   `json:"scrap_reason,omitempty"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedBy         uuid.UUID `json:"created_by"`
}

type AddConsumptionDTO struct {
	ProductionOrderID int64     `json:"production_order_id"`
	AppointmentID     *int64    `json:"appointment_id,omitempty"`
	ItemCode          int64     `json:"item_code"`
	ConsumedQty       float64   `json:"consumed_qty"`
	WarehouseID       *int64    `json:"warehouse_id,omitempty"`
	Lot               *string   `json:"lot,omitempty"`
	ConsumptionDate   string    `json:"consumption_date"`
	Notes             *string   `json:"notes,omitempty"`
	CreatedBy         uuid.UUID `json:"created_by"`
}

type CompleteProductionOrderDTO struct {
	ID      int64  `json:"id"`
	EndDate string `json:"end_date"`
}

type CloseProductionOrderDTO struct {
	ID int64 `json:"id"`
}

type CancelProductionOrderDTO struct {
	ID int64 `json:"id"`
}
