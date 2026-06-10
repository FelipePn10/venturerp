package response

import (
	"time"

	"github.com/google/uuid"
)

// PurchaseRequisitionResponse is the API representation of a purchase requisition.
type PurchaseRequisitionResponse struct {
	ID                    int64                             `json:"id"`
	Code                  int64                             `json:"code"`
	EnterpriseCode        int64                             `json:"enterprise_code"`
	RequestTypeCode       *int64                            `json:"request_type_code,omitempty"`
	RequesterEmployeeCode *int64                            `json:"requester_employee_code,omitempty"`
	EmissionDate          time.Time                         `json:"emission_date"`
	Status                string                            `json:"status"`
	Notes                 *string                           `json:"notes,omitempty"`
	IsActive              bool                              `json:"is_active"`
	CreatedAt             time.Time                         `json:"created_at"`
	CreatedBy             uuid.UUID                         `json:"created_by"`
	UpdatedAt             time.Time                         `json:"updated_at"`
	Items                 []PurchaseRequisitionItemResponse `json:"items,omitempty"`
}

// PurchaseRequisitionItemResponse is the API representation of a requisition line.
type PurchaseRequisitionItemResponse struct {
	ID                int64      `json:"id"`
	RequisitionCode   int64      `json:"requisition_code"`
	Sequence          int32      `json:"sequence"`
	ItemCode          int64      `json:"item_code"`
	Quantity          float64    `json:"quantity"`
	AttendedQty       float64    `json:"attended_qty"`
	CancelledQty      float64    `json:"cancelled_qty"`
	Balance           float64    `json:"balance"`
	UOM               *string    `json:"uom,omitempty"`
	CostCenterCode    *int64     `json:"cost_center_code,omitempty"`
	AccountingAccount *string    `json:"accounting_account,omitempty"`
	SuggestedPrice    float64    `json:"suggested_price"`
	DeliveryDate      *time.Time `json:"delivery_date,omitempty"`
	Application       *string    `json:"application,omitempty"`
	UtilizationType   *string    `json:"utilization_type,omitempty"`
	Status            string     `json:"status"`
	IsActive          bool       `json:"is_active"`
	CreatedAt         time.Time  `json:"created_at"`
}
