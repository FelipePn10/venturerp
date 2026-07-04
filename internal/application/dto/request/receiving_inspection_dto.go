package request

type CreateReceivingInspectionRouteDTO struct {
	EnterpriseCode        int64                        `json:"enterprise_code"`
	Basis                 string                       `json:"basis"`
	ItemCode              *int64                       `json:"item_code"`
	ClassificationCode    *string                      `json:"classification_code"`
	Mask                  string                       `json:"mask"`
	InspectionWarehouseID int64                        `json:"inspection_warehouse_id"`
	HandlingType          *string                      `json:"handling_type"`
	StorageType           *string                      `json:"storage_type"`
	RouteType             *string                      `json:"route_type"`
	MarketType            *string                      `json:"market_type"`
	InspectionType        *string                      `json:"inspection_type"`
	ValidFrom             string                       `json:"valid_from"`
	ValidTo               *string                      `json:"valid_to"`
	Steps                 []ReceivingInspectionStepDTO `json:"steps"`
}

type ReceivingInspectionStepDTO struct {
	Sequence        int                               `json:"sequence"`
	InspectionName  string                            `json:"inspection_name"`
	Kind            string                            `json:"kind"`
	AppointmentMode string                            `json:"appointment_mode"`
	IsRequired      bool                              `json:"is_required"`
	EmitsLabel      bool                              `json:"emits_label"`
	InstrumentGroup *string                           `json:"instrument_group"`
	SampleType      *string                           `json:"sample_type"`
	SampleUnit      *string                           `json:"sample_unit"`
	SampleQty       float64                           `json:"sample_qty"`
	AcceptanceQty   float64                           `json:"acceptance_qty"`
	RejectionQty    float64                           `json:"rejection_qty"`
	Norm            *string                           `json:"norm"`
	Reference       *string                           `json:"reference"`
	ValidTo         *string                           `json:"valid_to"`
	NominalValue    *float64                          `json:"nominal_value"`
	MinValue        *float64                          `json:"min_value"`
	MaxValue        *float64                          `json:"max_value"`
	Attributes      []ReceivingInspectionAttributeDTO `json:"attributes"`
}

type ReceivingInspectionAttributeDTO struct {
	Description string `json:"description"`
	IsApproved  bool   `json:"is_approved"`
}

type GenerateReceivingInspectionOrderDTO struct {
	Source                string  `json:"source"`
	SupplierCode          *int64  `json:"supplier_code"`
	PurchaseOrderCode     *int64  `json:"purchase_order_code"`
	PurchaseOrderItemCode *int64  `json:"purchase_order_item_code"`
	FiscalEntryCode       *int64  `json:"fiscal_entry_code"`
	ReceivingNoticeCode   *int64  `json:"receiving_notice_code"`
	ItemCode              int64   `json:"item_code"`
	ClassificationCode    *string `json:"classification_code"`
	Mask                  string  `json:"mask"`
	Lot                   *string `json:"lot"`
	SerialNumber          *string `json:"serial_number"`
	WarehouseID           int64   `json:"warehouse_id"`
	Quantity              float64 `json:"quantity"`
	Certificate           *string `json:"certificate"`
	SupplierNote          *string `json:"supplier_note"`
	Model                 *string `json:"model"`
	Notes                 *string `json:"notes"`
}

type RecordReceivingInspectionResultDTO struct {
	StepID               *int64   `json:"step_id"`
	Sequence             int      `json:"sequence"`
	SampleIndex          int      `json:"sample_index"`
	MeasuredValue        *float64 `json:"measured_value"`
	MinValue             *float64 `json:"min_value"`
	MaxValue             *float64 `json:"max_value"`
	AttributeDescription *string  `json:"attribute_description"`
	IsApproved           bool     `json:"is_approved"`
	Notes                *string  `json:"notes"`
}

type AnalyzeReceivingInspectionOrderDTO struct {
	ConformQty           float64 `json:"conform_qty"`
	RejectedQty          float64 `json:"rejected_qty"`
	ReworkQty            float64 `json:"rework_qty"`
	RestrictedQty        float64 `json:"restricted_qty"`
	Treatment            string  `json:"treatment"`
	AffectsSupplierScore bool    `json:"affects_supplier_score"`
	Notes                *string `json:"notes"`

	// Stock routing: when move_stock is true, the analyzed quantities leave the
	// inspection warehouse (order.warehouse_id) toward the destinations below.
	// This closes the inspection loop so approved material becomes available and
	// rejected/rework material is segregated, instead of only recording numbers.
	MoveStock              bool   `json:"move_stock"`
	DestinationWarehouseID *int64 `json:"destination_warehouse_id"`
	RejectionWarehouseID   *int64 `json:"rejection_warehouse_id"`
	ReworkWarehouseID      *int64 `json:"rework_warehouse_id"`
	RestrictedWarehouseID  *int64 `json:"restricted_warehouse_id"`
}
