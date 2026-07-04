package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type RecordType string
type RecordStatus string

const (
	RecordReceivingInspection RecordType = "RECEIVING_INSPECTION"
	RecordReceivingNotice     RecordType = "RECEIVING_NOTICE"
	RecordSupplierEvaluation  RecordType = "SUPPLIER_EVALUATION"
	RecordApprovalLimit       RecordType = "APPROVAL_LIMIT"
	RecordSupplierContract    RecordType = "SUPPLIER_CONTRACT"
	RecordReceivingChecklist  RecordType = "RECEIVING_CHECKLIST"
	RecordReceivingLabel      RecordType = "RECEIVING_LABEL"
	RecordSupplierEDI         RecordType = "SUPPLIER_EDI"
	RecordImportProcess       RecordType = "IMPORT_PROCESS"

	StatusDraft     RecordStatus = "DRAFT"
	StatusOpen      RecordStatus = "OPEN"
	StatusInReview  RecordStatus = "IN_REVIEW"
	StatusApproved  RecordStatus = "APPROVED"
	StatusRejected  RecordStatus = "REJECTED"
	StatusPartial   RecordStatus = "PARTIAL"
	StatusClosed    RecordStatus = "CLOSED"
	StatusCancelled RecordStatus = "CANCELLED"
)

type Record struct {
	ID                    int64
	RecordType            RecordType
	Status                RecordStatus
	SupplierCode          *int64
	PurchaseOrderCode     *int64
	PurchaseOrderItemCode *int64
	ItemCode              *int64
	Mask                  string
	WarehouseID           *int64
	Quantity              float64
	Reference             *string
	Payload               json.RawMessage
	OpenedAt              time.Time
	ClosedAt              *time.Time
	CreatedBy             *uuid.UUID
	UpdatedAt             time.Time
}

type InspectionDisposition struct {
	ID                     int64
	RecordID               int64
	ApprovedQty            float64
	RejectedQty            float64
	QuarantineWarehouseID  *int64
	DestinationWarehouseID *int64
	Reason                 *string
	DisposedAt             time.Time
	DisposedBy             *uuid.UUID
}

type SupplierScorecard struct {
	ID               int64
	SupplierCode     int64
	PeriodStart      time.Time
	PeriodEnd        time.Time
	QualityScore     float64
	DeliveryScore    float64
	CommercialScore  float64
	ServiceScore     float64
	OverallScore     float64
	TotalReceipts    int
	RejectedReceipts int
	LateReceipts     int
	Notes            *string
	CreatedAt        time.Time
	CreatedBy        *uuid.UUID
}

type ReceivingInspectionRoute struct {
	ID                    int64
	EnterpriseCode        int64
	Basis                 string
	ItemCode              *int64
	ClassificationCode    *string
	Mask                  string
	InspectionWarehouseID int64
	HandlingType          *string
	StorageType           *string
	RouteType             *string
	MarketType            *string
	InspectionType        *string
	ValidFrom             time.Time
	ValidTo               *time.Time
	IsActive              bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
	CreatedBy             *uuid.UUID
	Steps                 []*ReceivingInspectionRouteStep
}

type ReceivingInspectionRouteStep struct {
	ID              int64
	RouteID         int64
	Sequence        int
	InspectionName  string
	Kind            string
	AppointmentMode string
	IsRequired      bool
	EmitsLabel      bool
	InstrumentGroup *string
	SampleType      *string
	SampleUnit      *string
	SampleQty       float64
	AcceptanceQty   float64
	RejectionQty    float64
	Norm            *string
	Reference       *string
	ValidTo         *time.Time
	NominalValue    *float64
	MinValue        *float64
	MaxValue        *float64
	Attributes      []*ReceivingInspectionStepAttribute
}

type ReceivingInspectionStepAttribute struct {
	ID          int64
	StepID      int64
	Description string
	IsApproved  bool
}

type ReceivingInspectionOrder struct {
	ID                    int64
	OrderNumber           int64
	RouteID               *int64
	ProcurementRecordID   *int64
	Source                string
	SupplierCode          *int64
	PurchaseOrderCode     *int64
	PurchaseOrderItemCode *int64
	FiscalEntryCode       *int64
	ReceivingNoticeCode   *int64
	ItemCode              int64
	Mask                  string
	Lot                   *string
	SerialNumber          *string
	WarehouseID           int64
	Quantity              float64
	InspectedQty          float64
	ApprovedQty           float64
	RejectedQty           float64
	ReworkQty             float64
	RestrictedQty         float64
	Status                string
	Certificate           *string
	SupplierNote          *string
	Model                 *string
	Notes                 *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
	CreatedBy             *uuid.UUID
}

type ReceivingInspectionResult struct {
	ID                   int64
	OrderID              int64
	StepID               *int64
	Sequence             int
	SampleIndex          int
	MeasuredValue        *float64
	MinValue             *float64
	MaxValue             *float64
	AttributeDescription *string
	IsApproved           bool
	Notes                *string
	CreatedAt            time.Time
	CreatedBy            *uuid.UUID
}

type ReceivingInspectionAnalysis struct {
	ID                   int64
	OrderID              int64
	ConformQty           float64
	RejectedQty          float64
	ReworkQty            float64
	RestrictedQty        float64
	Treatment            string
	AffectsSupplierScore bool
	Notes                *string
	AnalyzedAt           time.Time
	AnalyzedBy           *uuid.UUID
	Order                *ReceivingInspectionOrder
}

// ApprovalLimit is a purchase approval rule (alçada de valores). A purchase order
// whose total is at or below AutoApproveMax is auto-approved; above it the order is
// blocked pending explicit authorization. BlockAbove, when set, is a hard ceiling
// that not even an authorizer can release.
type ApprovalLimit struct {
	ID             int64
	EnterpriseCode int64
	Scope          string // GLOBAL | SUPPLIER | COST_CENTER | CATEGORY
	ScopeRef       *string
	Currency       string
	AutoApproveMax float64
	BlockAbove     *float64
	IsActive       bool
	ValidFrom      time.Time
	ValidTo        *time.Time
	Notes          *string
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ApprovalDecision is the resolved outcome of evaluating a purchase amount against
// the applicable approval limit.
type ApprovalDecision struct {
	AutoApprove bool
	Blocked     bool     // over the hard ceiling; cannot be authorized
	Ceiling     float64  // the auto-approve ceiling that applied
	HardCeiling *float64 // block_above, when defined
	LimitID     *int64   // the rule that matched, if any
}

type SupplierContract struct {
	ID             int64
	EnterpriseCode int64
	SupplierCode   int64
	ContractNumber string
	Description    *string
	Status         string // DRAFT | ACTIVE | SUSPENDED | CLOSED | CANCELLED
	Currency       string
	ValidFrom      time.Time
	ValidTo        *time.Time
	PriceIndex     *string
	Notes          *string
	CreatedBy      *uuid.UUID
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Items          []*SupplierContractItem
}

type SupplierContractItem struct {
	ID            int64
	ContractID    int64
	ItemCode      int64
	Mask          string
	Unit          *string
	ContractedQty float64
	ConsumedQty   float64
	UnitPrice     float64
	MinOrderQty   float64
	Notes         *string
}

// RemainingQty is the still-available contracted balance for the item line.
func (i *SupplierContractItem) RemainingQty() float64 {
	remaining := i.ContractedQty - i.ConsumedQty
	if remaining < 0 {
		return 0
	}
	return remaining
}

// SupplierPerformanceAggregate holds the raw counters that feed IQF auto-computation
// for a supplier over a period, aggregated from receiving inspection orders/analyses
// and purchase order delivery data.
type SupplierPerformanceAggregate struct {
	TotalReceipts    int
	RejectedReceipts int
	LateReceipts     int
	InspectedQty     float64
	RejectedQty      float64
}

// PurchaseMovementHistoryRow is one aggregated line of the consolidated purchase
// movement history (buyer/supplier performance consult).
type PurchaseMovementHistoryRow struct {
	SupplierCode      *int64
	PurchaseOrderCode int64
	OrderNumber       int64
	ItemCode          int64
	Mask              string
	RequestedQty      float64
	ReceivedQty       float64
	CancelledQty      float64
	UnitPrice         float64
	Status            string
	EmissionDate      time.Time
	DeliveryDate      *time.Time
}
