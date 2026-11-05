package entity

import (
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"github.com/shopspring/decimal"
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
		return nil, domainerr.NewValidation("informe o código")
	}
	if enterpriseCode == 0 {
		return nil, domainerr.NewValidation("informe a empresa")
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
	Quantity          decimal.Decimal
	AttendedQty       decimal.Decimal
	CancelledQty      decimal.Decimal
	UOM               *string
	CostCenterCode    *int64
	AccountingAccount *string
	SuggestedPrice    decimal.Decimal
	DeliveryDate      *time.Time
	WarehouseID       *int64
	Application       *string
	UtilizationType   *string
	Status            RequisitionStatus
	IsActive          bool
	CreatedAt         time.Time
}

// Balance is the still-open quantity (Qtde Saldo).
func (i *PurchaseRequisitionItem) Balance() decimal.Decimal {
	b := i.Quantity.Sub(i.AttendedQty).Sub(i.CancelledQty)
	if b.IsNegative() {
		return decimal.Zero
	}
	return b
}
