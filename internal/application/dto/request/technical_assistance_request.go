package request

import "github.com/google/uuid"

type CreateTADefectGroupDTO struct {
	Description string    `json:"description"`
	CreatedBy   uuid.UUID `json:"created_by"`
}

type CreateTADefectReasonDTO struct {
	GroupCode                int64     `json:"group_code"`
	Description              string    `json:"description"`
	AllowsComplement         bool      `json:"allows_complement"`
	GeneratesRevenue         bool      `json:"generates_revenue"`
	RequiresReturnNote       bool      `json:"requires_return_note"`
	GeneratesSalesOrder      bool      `json:"generates_sales_order"`
	GeneratesProductionOrder bool      `json:"generates_production_order"`
	IsReplacement            bool      `json:"is_replacement"`
	IsService                bool      `json:"is_service"`
	AvailableWeb             bool      `json:"available_web"`
	CreatedBy                uuid.UUID `json:"created_by"`
}

type CreateTAWarrantyResponsibleDTO struct {
	Name         string    `json:"name"`
	EmployeeCode *int64    `json:"employee_code"`
	CustomerCode *int64    `json:"customer_code"`
	Email        *string   `json:"email"`
	Phone        *string   `json:"phone"`
	CreatedBy    uuid.UUID `json:"created_by"`
}

type CreateTechnicalAssistanceCallDTO struct {
	EnterpriseCode          int64                                  `json:"enterprise_code"`
	CustomerCode            int64                                  `json:"customer_code"`
	ConsumerName            *string                                `json:"consumer_name"`
	ConsumerDocument        *string                                `json:"consumer_document"`
	TechnicalAssistantCode  *int64                                 `json:"technical_assistant_code"`
	WarrantyResponsibleCode *int64                                 `json:"warranty_responsible_code"`
	Priority                string                                 `json:"priority"`
	OpenedAt                string                                 `json:"opened_at"`
	PromisedDate            string                                 `json:"promised_date"`
	Subject                 string                                 `json:"subject"`
	Description             *string                                `json:"description"`
	ReturnNoteRequired      bool                                   `json:"return_note_required"`
	CreatedBy               uuid.UUID                              `json:"created_by"`
	Items                   []CreateTechnicalAssistanceCallItemDTO `json:"items"`
}

type CreateTechnicalAssistanceCallItemDTO struct {
	CallCode              int64   `json:"call_code"`
	Sequence              int     `json:"sequence"`
	ItemCode              int64   `json:"item_code"`
	Mask                  string  `json:"mask"`
	SerialNumber          *string `json:"serial_number"`
	Quantity              float64 `json:"quantity"`
	DefectReasonCode      *int64  `json:"defect_reason_code"`
	DefectComplement      *string `json:"defect_complement"`
	PurchaseInvoiceNumber *string `json:"purchase_invoice_number"`
	PurchaseInvoiceDate   string  `json:"purchase_invoice_date"`
	WarrantyDays          int     `json:"warranty_days"`
	RequestedAction       string  `json:"requested_action"`
	Notes                 *string `json:"notes"`
}

type AddTechnicalAssistanceReturnNoteDTO struct {
	CallCode      int64   `json:"call_code"`
	NoteNumber    string  `json:"note_number"`
	NoteSeries    *string `json:"note_series"`
	EmissionDate  string  `json:"emission_date"`
	CustomerCode  *int64  `json:"customer_code"`
	OperationType string  `json:"operation_type"`
	AccessKey     *string `json:"access_key"`
	TotalValue    float64 `json:"total_value"`
	Notes         *string `json:"notes"`
}

type UpdateTechnicalAssistanceCallStatusDTO struct {
	Code                 int64   `json:"code"`
	Status               string  `json:"status"`
	Diagnosis            *string `json:"diagnosis"`
	Solution             *string `json:"solution"`
	ServiceInvoiceNumber *string `json:"service_invoice_number"`
	CloseReason          *string `json:"close_reason"`
}

type CreateTechnicalAssistanceRMADTO struct {
	CallCode          int64                           `json:"-"`
	ReasonCode        string                          `json:"reason_code"`
	ReasonDescription *string                         `json:"reason_description"`
	EligibilityStatus string                          `json:"eligibility_status"`
	EligibilityReason string                          `json:"eligibility_reason"`
	SLADueAt          string                          `json:"sla_due_at"`
	ProductCost       float64                         `json:"product_cost"`
	FreightCost       float64                         `json:"freight_cost"`
	ServiceCost       float64                         `json:"service_cost"`
	Items             []TechnicalAssistanceRMAItemDTO `json:"items"`
	IdempotencyKey    string                          `json:"-"`
}

type TechnicalAssistanceRMAItemDTO struct {
	CallItemCode         int64   `json:"call_item_code"`
	ItemCode             int64   `json:"item_code"`
	Quantity             float64 `json:"quantity"`
	SerialNumber         *string `json:"serial_number"`
	LotNumber            *string `json:"lot_number"`
	RequestedDestination *string `json:"requested_destination"`
}

type TransitionTechnicalAssistanceRMADTO struct {
	Status              string   `json:"status"`
	Reason              *string  `json:"reason"`
	CorrelationID       *string  `json:"correlation_id"`
	AuthorizationNumber *string  `json:"authorization_number"`
	ReverseCarrierCode  *int64   `json:"reverse_carrier_code"`
	ReverseTrackingCode *string  `json:"reverse_tracking_code"`
	InspectionNotes     *string  `json:"inspection_notes"`
	Destination         *string  `json:"destination"`
	FiscalDocumentKey   *string  `json:"fiscal_document_key"`
	StockMovementCode   *int64   `json:"stock_movement_code"`
	ProductCost         *float64 `json:"product_cost"`
	FreightCost         *float64 `json:"freight_cost"`
	ServiceCost         *float64 `json:"service_cost"`
}

type GenerateTechnicalAssistanceOrdersDTO struct {
	CallCode          int64  `json:"call_code"`
	SalesDivisionCode *int64 `json:"sales_division_code"`
	PriceTableCode    *int64 `json:"price_table_code"`
	PaymentTermCode   *int64 `json:"payment_term_code"`
	WarehouseCode     *int64 `json:"warehouse_code"`
}
