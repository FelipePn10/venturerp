package response

import (
	"time"

	"github.com/google/uuid"
)

// PlannedOrderResponse is the API representation of a planned order.
type PlannedOrderResponse struct {
	Code              int64      `json:"code"`
	OrderNumber       int64      `json:"order_number"`
	ItemCode          int64      `json:"item_code"`
	Mask              *string    `json:"mask,omitempty"`
	Quantity          float64    `json:"quantity"`
	QuantityLoss      float64    `json:"quantity_loss"`
	QuantityCorrected float64    `json:"quantity_corrected"`
	OrderType         string     `json:"order_type"`
	Status            string     `json:"status"`
	PlanCode          *int64     `json:"plan_code,omitempty"`
	DemandType        string     `json:"demand_type"`
	DemandCode        *int64     `json:"demand_code,omitempty"`
	NeedDate          time.Time  `json:"need_date"`
	StartDate         *time.Time `json:"start_date,omitempty"`
	EndDate           *time.Time `json:"end_date,omitempty"`
	CostCenterCode    *int64     `json:"cost_center_code,omitempty"`
	EmployeeCode      *int64     `json:"employee_code,omitempty"`
	MachineCode       *int64     `json:"machine_code,omitempty"`
	ProductionTime    float64    `json:"production_time"`
	Priority          *string    `json:"priority,omitempty"`
	LLC               int        `json:"llc"`
	Notes             *string    `json:"notes,omitempty"`
	ParentOrderCode   *int64     `json:"parent_order_code,omitempty"`
	SalesOrderCode    *int64     `json:"sales_order_code,omitempty"`
	IsFirm            bool       `json:"is_firm"`
	IsActive          bool       `json:"is_active"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	CreatedBy         uuid.UUID  `json:"created_by"`
}
