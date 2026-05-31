package request

import "github.com/google/uuid"

type RequisitionItemInput struct {
	ItemCode          int64   `json:"item_code"`
	Quantity          float64 `json:"quantity"`
	UOM               *string `json:"uom,omitempty"`
	CostCenterCode    *int64  `json:"cost_center_code,omitempty"`
	AccountingAccount *string `json:"accounting_account,omitempty"`
	SuggestedPrice    float64 `json:"suggested_price,omitempty"`
	DeliveryDate      *string `json:"delivery_date,omitempty"`
	Application       *string `json:"application,omitempty"`
	UtilizationType   *string `json:"utilization_type,omitempty"`
}

type CreatePurchaseRequisitionDTO struct {
	EnterpriseCode        int64                  `json:"enterprise_code"`
	RequestTypeCode       *int64                 `json:"request_type_code,omitempty"`
	RequesterEmployeeCode *int64                 `json:"requester_employee_code,omitempty"`
	Notes                 *string                `json:"notes,omitempty"`
	CreatedBy             uuid.UUID              `json:"created_by"`
	Items                 []RequisitionItemInput `json:"items,omitempty"`
}

type AddRequisitionItemDTO struct {
	RequisitionCode int64 `json:"-"`
	RequisitionItemInput
}

// ─── Geração de Pedidos a partir de Solicitações ───────────────────────────────

type GenerationSelection struct {
	RequisitionItemID int64   `json:"requisition_item_id"`
	QtyToAttend       float64 `json:"qty_to_attend"`
	// SupplierCode optional; when absent, the item's preferred supplier is used.
	SupplierCode *int64 `json:"supplier_code,omitempty"`
}

type GeneratePurchaseOrdersDTO struct {
	EnterpriseCode int64                 `json:"enterprise_code"`
	CreatedBy      uuid.UUID             `json:"created_by"`
	Selections     []GenerationSelection `json:"selections"`
}
