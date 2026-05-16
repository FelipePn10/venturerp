package entity

import (
	"time"

	"github.com/google/uuid"
)

type PurchaseOrderStatus string

const (
	PurchaseOrderStatusDRAFT     PurchaseOrderStatus = "DRAFT"
	PurchaseOrderStatusREQUESTED PurchaseOrderStatus = "REQUESTED"
	PurchaseOrderStatusAPPROVED  PurchaseOrderStatus = "APPROVED"
	PurchaseOrderStatusPARTIAL   PurchaseOrderStatus = "PARTIAL"
	PurchaseOrderStatusRECEIVED  PurchaseOrderStatus = "RECEIVED"
	PurchaseOrderStatusCANCELLED PurchaseOrderStatus = "CANCELLED"
)

type PurchaseOrderOrigin string

const (
	PurchaseOrderOriginNORMAL       PurchaseOrderOrigin = "NORMAL"
	PurchaseOrderOriginMRP          PurchaseOrderOrigin = "MRP"
	PurchaseOrderOriginMANUAL       PurchaseOrderOrigin = "MANUAL"
	PurchaseOrderOriginINTERFABRICA PurchaseOrderOrigin = "INTERFABRICA"
)

type PurchaseOrderItemStatus string

const (
	PurchaseOrderItemStatusOPEN      PurchaseOrderItemStatus = "OPEN"
	PurchaseOrderItemStatusPARTIAL   PurchaseOrderItemStatus = "PARTIAL"
	PurchaseOrderItemStatusRECEIVED  PurchaseOrderItemStatus = "RECEIVED"
	PurchaseOrderItemStatusCANCELLED PurchaseOrderItemStatus = "CANCELLED"
)

type PurchaseOrder struct {
	Code               int64
	OrderNumber        int64
	EnterpriseCode     int64
	Status             PurchaseOrderStatus
	Origin             PurchaseOrderOrigin
	EmissionDate       time.Time
	DeliveryDate       *time.Time
	SupplierCode       *int64
	PaymentTermCode    *int64
	CurrencyCode       string
	ShippingAddressCode *int64
	Notes              *string

	TotalGross    float64
	TotalNet      float64
	TotalDiscount float64

	IsActive  bool
	IsFirm    bool
	CreatedAt time.Time
	UpdatedAt time.Time
	CreatedBy uuid.UUID

	Items []*PurchaseOrderItem
}

type PurchaseOrderItem struct {
	Code              int64
	PurchaseOrderCode int64
	Sequence          int
	ItemCode          int64
	Mask              string
	RequestedQty      float64
	ReceivedQty       float64
	CancelledQty      float64
	UnitPrice         float64
	TotalPrice        float64
	DiscountPct       float64
	IPIPct            float64
	ICMSPct           float64
	Status            PurchaseOrderItemStatus
	DeliveryDate      *time.Time
	Notes             *string
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
