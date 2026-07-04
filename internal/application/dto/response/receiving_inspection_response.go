package response

import (
	"time"

	"github.com/google/uuid"
)

type ReceivingInspectionRouteResponse struct {
	ID                    int64                             `json:"id"`
	EnterpriseCode        int64                             `json:"enterprise_code"`
	Basis                 string                            `json:"basis"`
	ItemCode              *int64                            `json:"item_code,omitempty"`
	ClassificationCode    *string                           `json:"classification_code,omitempty"`
	Mask                  string                            `json:"mask"`
	InspectionWarehouseID int64                             `json:"inspection_warehouse_id"`
	HandlingType          *string                           `json:"handling_type,omitempty"`
	StorageType           *string                           `json:"storage_type,omitempty"`
	RouteType             *string                           `json:"route_type,omitempty"`
	MarketType            *string                           `json:"market_type,omitempty"`
	InspectionType        *string                           `json:"inspection_type,omitempty"`
	ValidFrom             time.Time                         `json:"valid_from"`
	ValidTo               *time.Time                        `json:"valid_to,omitempty"`
	IsActive              bool                              `json:"is_active"`
	CreatedAt             time.Time                         `json:"created_at"`
	CreatedBy             *uuid.UUID                        `json:"created_by,omitempty"`
	Steps                 []ReceivingInspectionStepResponse `json:"steps"`
}

type ReceivingInspectionStepResponse struct {
	ID              int64                                  `json:"id"`
	RouteID         int64                                  `json:"route_id"`
	Sequence        int                                    `json:"sequence"`
	InspectionName  string                                 `json:"inspection_name"`
	Kind            string                                 `json:"kind"`
	AppointmentMode string                                 `json:"appointment_mode"`
	IsRequired      bool                                   `json:"is_required"`
	EmitsLabel      bool                                   `json:"emits_label"`
	InstrumentGroup *string                                `json:"instrument_group,omitempty"`
	SampleType      *string                                `json:"sample_type,omitempty"`
	SampleUnit      *string                                `json:"sample_unit,omitempty"`
	SampleQty       float64                                `json:"sample_qty"`
	AcceptanceQty   float64                                `json:"acceptance_qty"`
	RejectionQty    float64                                `json:"rejection_qty"`
	Norm            *string                                `json:"norm,omitempty"`
	Reference       *string                                `json:"reference,omitempty"`
	ValidTo         *time.Time                             `json:"valid_to,omitempty"`
	NominalValue    *float64                               `json:"nominal_value,omitempty"`
	MinValue        *float64                               `json:"min_value,omitempty"`
	MaxValue        *float64                               `json:"max_value,omitempty"`
	Attributes      []ReceivingInspectionAttributeResponse `json:"attributes"`
}

type ReceivingInspectionAttributeResponse struct {
	ID          int64  `json:"id"`
	StepID      int64  `json:"step_id"`
	Description string `json:"description"`
	IsApproved  bool   `json:"is_approved"`
}

type ReceivingInspectionOrderResponse struct {
	ID                    int64      `json:"id"`
	OrderNumber           int64      `json:"order_number"`
	RouteID               *int64     `json:"route_id,omitempty"`
	ProcurementRecordID   *int64     `json:"procurement_record_id,omitempty"`
	Source                string     `json:"source"`
	SupplierCode          *int64     `json:"supplier_code,omitempty"`
	PurchaseOrderCode     *int64     `json:"purchase_order_code,omitempty"`
	PurchaseOrderItemCode *int64     `json:"purchase_order_item_code,omitempty"`
	FiscalEntryCode       *int64     `json:"fiscal_entry_code,omitempty"`
	ReceivingNoticeCode   *int64     `json:"receiving_notice_code,omitempty"`
	ItemCode              int64      `json:"item_code"`
	Mask                  string     `json:"mask"`
	Lot                   *string    `json:"lot,omitempty"`
	SerialNumber          *string    `json:"serial_number,omitempty"`
	WarehouseID           int64      `json:"warehouse_id"`
	Quantity              float64    `json:"quantity"`
	InspectedQty          float64    `json:"inspected_qty"`
	ApprovedQty           float64    `json:"approved_qty"`
	RejectedQty           float64    `json:"rejected_qty"`
	ReworkQty             float64    `json:"rework_qty"`
	RestrictedQty         float64    `json:"restricted_qty"`
	Status                string     `json:"status"`
	Certificate           *string    `json:"certificate,omitempty"`
	SupplierNote          *string    `json:"supplier_note,omitempty"`
	Model                 *string    `json:"model,omitempty"`
	Notes                 *string    `json:"notes,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	CreatedBy             *uuid.UUID `json:"created_by,omitempty"`
}

type ReceivingInspectionResultResponse struct {
	ID                   int64      `json:"id"`
	OrderID              int64      `json:"order_id"`
	StepID               *int64     `json:"step_id,omitempty"`
	Sequence             int        `json:"sequence"`
	SampleIndex          int        `json:"sample_index"`
	MeasuredValue        *float64   `json:"measured_value,omitempty"`
	MinValue             *float64   `json:"min_value,omitempty"`
	MaxValue             *float64   `json:"max_value,omitempty"`
	AttributeDescription *string    `json:"attribute_description,omitempty"`
	IsApproved           bool       `json:"is_approved"`
	Notes                *string    `json:"notes,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	CreatedBy            *uuid.UUID `json:"created_by,omitempty"`
}

type ReceivingInspectionAnalysisResponse struct {
	ID                   int64                             `json:"id"`
	OrderID              int64                             `json:"order_id"`
	ConformQty           float64                           `json:"conform_qty"`
	RejectedQty          float64                           `json:"rejected_qty"`
	ReworkQty            float64                           `json:"rework_qty"`
	RestrictedQty        float64                           `json:"restricted_qty"`
	Treatment            string                            `json:"treatment"`
	AffectsSupplierScore bool                              `json:"affects_supplier_score"`
	Notes                *string                           `json:"notes,omitempty"`
	AnalyzedAt           time.Time                         `json:"analyzed_at"`
	AnalyzedBy           *uuid.UUID                        `json:"analyzed_by,omitempty"`
	Order                *ReceivingInspectionOrderResponse `json:"order"`
	Movements            []StockMovementResponse           `json:"movements,omitempty"`
}
