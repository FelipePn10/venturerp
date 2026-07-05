package response

import (
	"time"

	tarepo "github.com/FelipePn10/panossoerp/internal/domain/technical_assistance/repository"
)

type TADefectGroupResponse struct {
	Code        int64     `json:"code"`
	Description string    `json:"description"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type TADefectReasonResponse struct {
	Code                     int64     `json:"code"`
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
	IsActive                 bool      `json:"is_active"`
	CreatedAt                time.Time `json:"created_at"`
}

type TAWarrantyResponsibleResponse struct {
	Code         int64     `json:"code"`
	Name         string    `json:"name"`
	EmployeeCode *int64    `json:"employee_code,omitempty"`
	CustomerCode *int64    `json:"customer_code,omitempty"`
	Email        *string   `json:"email,omitempty"`
	Phone        *string   `json:"phone,omitempty"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}

type TechnicalAssistanceCallResponse struct {
	Code                 int64                                   `json:"code"`
	CallNumber           int64                                   `json:"call_number"`
	EnterpriseCode       int64                                   `json:"enterprise_code"`
	CustomerCode         int64                                   `json:"customer_code"`
	Status               string                                  `json:"status"`
	Priority             string                                  `json:"priority"`
	OpenedAt             time.Time                               `json:"opened_at"`
	PromisedDate         *time.Time                              `json:"promised_date,omitempty"`
	AttendedAt           *time.Time                              `json:"attended_at,omitempty"`
	ClosedAt             *time.Time                              `json:"closed_at,omitempty"`
	Subject              string                                  `json:"subject"`
	Description          *string                                 `json:"description,omitempty"`
	Diagnosis            *string                                 `json:"diagnosis,omitempty"`
	Solution             *string                                 `json:"solution,omitempty"`
	ReturnNoteRequired   bool                                    `json:"return_note_required"`
	SalesOrderCode       *int64                                  `json:"sales_order_code,omitempty"`
	ProductionOrderID    *int64                                  `json:"production_order_id,omitempty"`
	ServiceInvoiceNumber *string                                 `json:"service_invoice_number,omitempty"`
	CloseReason          *string                                 `json:"close_reason,omitempty"`
	Items                []TechnicalAssistanceCallItemResponse   `json:"items,omitempty"`
	ReturnNotes          []TechnicalAssistanceReturnNoteResponse `json:"return_notes,omitempty"`
}

type TechnicalAssistanceCallItemResponse struct {
	Code             int64      `json:"code"`
	Sequence         int        `json:"sequence"`
	ItemCode         int64      `json:"item_code"`
	Mask             string     `json:"mask"`
	Quantity         float64    `json:"quantity"`
	DefectReasonCode *int64     `json:"defect_reason_code,omitempty"`
	WarrantyUntil    *time.Time `json:"warranty_until,omitempty"`
	InWarranty       bool       `json:"in_warranty"`
	GeneratesRevenue bool       `json:"generates_revenue"`
	RequestedAction  string     `json:"requested_action"`
	Status           string     `json:"status"`
}

type TechnicalAssistanceReturnNoteResponse struct {
	Code          int64     `json:"code"`
	CallCode      int64     `json:"call_code"`
	NoteNumber    string    `json:"note_number"`
	EmissionDate  time.Time `json:"emission_date"`
	OperationType string    `json:"operation_type"`
	TotalValue    float64   `json:"total_value"`
}

type TechnicalAssistanceOrderGenerationResponse struct {
	CallCode          int64  `json:"call_code"`
	SalesOrderCode    *int64 `json:"sales_order_code,omitempty"`
	ProductionOrderID *int64 `json:"production_order_id,omitempty"`
	GeneratedLinks    int    `json:"generated_links"`
}

type TechnicalAssistanceReportResponse = tarepo.Report
