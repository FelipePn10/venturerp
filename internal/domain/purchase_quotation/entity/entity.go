package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type QuotationStatus string

const (
	QuotationOpen      QuotationStatus = "OPEN"
	QuotationQuoted    QuotationStatus = "QUOTED"
	QuotationClosed    QuotationStatus = "CLOSED"
	QuotationCancelled QuotationStatus = "CANCELLED"
)

type QuotationSourceType string

const (
	SourceRequisition  QuotationSourceType = "REQUISITION"
	SourcePlannedOrder QuotationSourceType = "PLANNED_ORDER"
	SourceManual       QuotationSourceType = "MANUAL"
)

type PurchaseQuotation struct {
	ID             int64
	Code           int64
	EnterpriseCode int64
	Status         QuotationStatus
	EmissionDate   time.Time
	Notes          *string
	IsActive       bool
	CreatedAt      time.Time
	CreatedBy      uuid.UUID
	UpdatedAt      time.Time
	Items          []*PurchaseQuotationItem
	Suppliers      []*PurchaseQuotationSupplier
}

func NewPurchaseQuotation(code, enterpriseCode int64, createdBy uuid.UUID) (*PurchaseQuotation, error) {
	if code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	if enterpriseCode == 0 {
		return nil, fmt.Errorf("enterprise_code is required")
	}
	now := time.Now()
	return &PurchaseQuotation{
		Code:           code,
		EnterpriseCode: enterpriseCode,
		Status:         QuotationOpen,
		EmissionDate:   now,
		IsActive:       true,
		CreatedAt:      now,
		CreatedBy:      createdBy,
		UpdatedAt:      now,
	}, nil
}

type PurchaseQuotationItem struct {
	ID            int64
	QuotationCode int64
	Sequence      int32
	ItemCode      int64
	Quantity      float64
	UOM           *string
	DeliveryDate  *time.Time
	SourceType    QuotationSourceType
	SourceCode    *int64
	SourceItemID  *int64
	IsConfigured  bool
	CreatedAt     time.Time
	Prices        []*PurchaseQuotationPrice
}

type PurchaseQuotationSupplier struct {
	ID            int64
	QuotationCode int64
	SupplierCode  int64
	InvitedAt     time.Time
}

type PurchaseQuotationPrice struct {
	ID              int64
	QuotationItemID int64
	SupplierCode    int64
	UnitPrice       float64
	LeadTimeDays    int32
	PaymentTermCode *int64
	Notes           *string
	IsSelected      bool
	CreatedAt       time.Time
}
