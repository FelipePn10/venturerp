package entity

import (
	"time"

	"github.com/google/uuid"
)

type CallStatus string

const (
	CallStatusPending       CallStatus = "PENDING"
	CallStatusInAnalysis    CallStatus = "IN_ANALYSIS"
	CallStatusWaitingReturn CallStatus = "WAITING_RETURN"
	CallStatusWaitingOrder  CallStatus = "WAITING_ORDER"
	CallStatusAttended      CallStatus = "ATTENDED"
	CallStatusClosed        CallStatus = "CLOSED"
	CallStatusCancelled     CallStatus = "CANCELLED"
)

type DefectGroup struct {
	Code        int64
	Description string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   uuid.UUID
}

type DefectReason struct {
	Code                     int64
	GroupCode                int64
	Description              string
	AllowsComplement         bool
	GeneratesRevenue         bool
	RequiresReturnNote       bool
	GeneratesSalesOrder      bool
	GeneratesProductionOrder bool
	IsReplacement            bool
	IsService                bool
	AvailableWeb             bool
	IsActive                 bool
	CreatedAt                time.Time
	UpdatedAt                time.Time
	CreatedBy                uuid.UUID
}

type WarrantyResponsible struct {
	Code         int64
	Name         string
	EmployeeCode *int64
	CustomerCode *int64
	Email        *string
	Phone        *string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedBy    uuid.UUID
}

type Call struct {
	Code                    int64
	CallNumber              int64
	EnterpriseCode          int64
	CustomerCode            int64
	ConsumerName            *string
	ConsumerDocument        *string
	TechnicalAssistantCode  *int64
	WarrantyResponsibleCode *int64
	Status                  CallStatus
	Priority                string
	OpenedAt                time.Time
	PromisedDate            *time.Time
	AttendedAt              *time.Time
	ClosedAt                *time.Time
	Subject                 string
	Description             *string
	Diagnosis               *string
	Solution                *string
	ReturnNoteRequired      bool
	SalesOrderCode          *int64
	ProductionOrderID       *int64
	ServiceInvoiceNumber    *string
	CloseReason             *string
	IsActive                bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
	CreatedBy               uuid.UUID
	Items                   []*CallItem
	ReturnNotes             []*ReturnNote
}

type CallItem struct {
	Code                  int64
	CallCode              int64
	Sequence              int
	ItemCode              int64
	Mask                  string
	SerialNumber          *string
	Quantity              float64
	DefectReasonCode      *int64
	DefectComplement      *string
	PurchaseInvoiceNumber *string
	PurchaseInvoiceDate   *time.Time
	WarrantyDays          int
	WarrantyUntil         *time.Time
	InWarranty            bool
	GeneratesRevenue      bool
	RequestedAction       string
	Status                string
	Notes                 *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type ReturnNote struct {
	Code          int64
	CallCode      int64
	NoteNumber    string
	NoteSeries    *string
	EmissionDate  time.Time
	CustomerCode  *int64
	OperationType string
	AccessKey     *string
	TotalValue    float64
	Notes         *string
	CreatedAt     time.Time
	CreatedBy     uuid.UUID
}

type OrderLink struct {
	Code              int64
	CallCode          int64
	CallItemCode      *int64
	GeneratedType     string
	SalesOrderCode    *int64
	ProductionOrderID *int64
	GeneratedAt       time.Time
	CreatedBy         uuid.UUID
	Notes             *string
}
