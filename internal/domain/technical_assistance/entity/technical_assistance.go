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

type RMA struct {
	Code                int64
	EnterpriseID        int64
	CallCode            int64
	Status              string
	ReasonCode          string
	ReasonDescription   *string
	EligibilityStatus   string
	EligibilityReason   string
	AuthorizationNumber *string
	AuthorizedAt        *time.Time
	ReverseCarrierCode  *int64
	ReverseTrackingCode *string
	ReceivedAt          *time.Time
	InspectionNotes     *string
	InspectedAt         *time.Time
	Destination         *string
	SLADueAt            time.Time
	ProductCost         float64
	FreightCost         float64
	ServiceCost         float64
	FiscalDocumentKey   *string
	StockMovementCode   *int64
	IdempotencyKey      string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	CreatedBy           uuid.UUID
	Items               []*RMAItem
	Events              []*RMAEvent
	Evidences           []*RMAEvidence
}

type RMAEvidence struct {
	ID          uuid.UUID
	RMACode     int64
	FileName    string
	ContentType string
	Content     []byte
	SizeBytes   int64
	SHA256      string
	UploadedBy  uuid.UUID
	CreatedAt   time.Time
}

type RMAItem struct {
	Code                 int64
	RMACode              int64
	CallItemCode         int64
	ItemCode             int64
	Quantity             float64
	SerialNumber         *string
	LotNumber            *string
	RequestedDestination *string
	InspectionResult     *string
}

type RMAEvent struct {
	Code          int64
	RMACode       int64
	EventType     string
	BeforeState   []byte
	AfterState    []byte
	Reason        *string
	CorrelationID *string
	OccurredAt    time.Time
	ActorID       uuid.UUID
}
