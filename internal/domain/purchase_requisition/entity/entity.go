package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type RequisitionStatus string

const (
	ReqStatusOpen      RequisitionStatus = "OPEN"
	ReqStatusPartial   RequisitionStatus = "PARTIAL"
	ReqStatusAttended  RequisitionStatus = "ATTENDED"
	ReqStatusCancelled RequisitionStatus = "CANCELLED"
)

type PurchaseRequisition struct {
	ID                    int64
	Code                  int64
	EnterpriseCode        int64
	RequestTypeCode       *int64
	RequesterEmployeeCode *int64
	EmissionDate          time.Time
	Status                RequisitionStatus
	Notes                 *string
	IsActive              bool
	CreatedAt             time.Time
	CreatedBy             uuid.UUID
	UpdatedAt             time.Time
	Items                 []*PurchaseRequisitionItem
}

func NewPurchaseRequisition(code, enterpriseCode int64, createdBy uuid.UUID) (*PurchaseRequisition, error) {
	if code == 0 {
		return nil, fmt.Errorf("code is required")
	}
	if enterpriseCode == 0 {
		return nil, fmt.Errorf("enterprise_code is required")
	}
	now := time.Now()
	return &PurchaseRequisition{
		Code:           code,
		EnterpriseCode: enterpriseCode,
		EmissionDate:   now,
		Status:         ReqStatusOpen,
		IsActive:       true,
		CreatedAt:      now,
		CreatedBy:      createdBy,
		UpdatedAt:      now,
	}, nil
}

type PurchaseRequisitionItem struct {
	ID                int64
	RequisitionCode   int64
	Sequence          int32
	ItemCode          int64
	Quantity          float64
	AttendedQty       float64
	CancelledQty      float64
	UOM               *string
	CostCenterCode    *int64
	AccountingAccount *string
	SuggestedPrice    float64
	DeliveryDate      *time.Time
	Application       *string
	UtilizationType   *string
	Status            RequisitionStatus
	IsActive          bool
	CreatedAt         time.Time
}

// Balance is the still-open quantity (Qtde Saldo).
func (i *PurchaseRequisitionItem) Balance() float64 {
	b := i.Quantity - i.AttendedQty - i.CancelledQty
	if b < 0 {
		return 0
	}
	return b
}
