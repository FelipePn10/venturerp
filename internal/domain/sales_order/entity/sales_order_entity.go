package entity

import (
	"time"

	"github.com/google/uuid"
)

// SalesOrderStatus represents the lifecycle status of a sales order.
type SalesOrderStatus string

const (
	SalesOrderStatusDraft          SalesOrderStatus = "R"  // Rascunho
	SalesOrderStatusOrder          SalesOrderStatus = "P"  // Pedido
	SalesOrderStatusAnalysis       SalesOrderStatus = "A"  // Pedido em Análise
	SalesOrderStatusBudgetAnalysis SalesOrderStatus = "OA" // Orçamento em Análise
	SalesOrderStatusBudget         SalesOrderStatus = "OF" // Orçamento
	SalesOrderStatusInvoiced       SalesOrderStatus = "F"  // Faturado (NF-e de saída autorizada)
	SalesOrderStatusCancelled      SalesOrderStatus = "CANCELLED"
)

// SalesOrderOrigin represents how the order was originated.
type SalesOrderOrigin string

const (
	SalesOrderOriginNormal     SalesOrderOrigin = "NORMAL"
	SalesOrderOriginDependent  SalesOrderOrigin = "DEPENDENT"
	SalesOrderOriginAssistance SalesOrderOrigin = "ASSISTANCE"
	SalesOrderOriginReserve    SalesOrderOrigin = "RESERVE"
	SalesOrderOriginCopy       SalesOrderOrigin = "COPY"
)

// SalesOrderItemStatus represents the fulfillment status of an order line.
type SalesOrderItemStatus string

const (
	SalesOrderItemStatusOpen      SalesOrderItemStatus = "OPEN"
	SalesOrderItemStatusPartial   SalesOrderItemStatus = "PARTIAL"
	SalesOrderItemStatusDelivered SalesOrderItemStatus = "DELIVERED"
	SalesOrderItemStatusCancelled SalesOrderItemStatus = "CANCELLED"
)

type SalesOrder struct {
	Code                        int64
	OrderNumber                 int64
	EnterpriseCode              int64
	Status                      SalesOrderStatus
	Origin                      SalesOrderOrigin
	EmissionDate                time.Time
	DeliveryDate                *time.Time
	DeliveryDateFirm            bool
	DigitDate                   time.Time
	CustomerCode                *int64
	BillingAddressCode          *int64
	ShippingAddressCode         *int64
	RepresentativeCode          *int64
	RepresentativeOrderNumber   *int64
	PlanCode                    *int64
	SalesDivisionCode           *int64
	CommissionPct               float64
	TaxTypeCode                 *int64
	PresenceIndicator           *string
	SalesChannel                *string
	DefaultNFType               *string
	PriceTableCode              *int64
	CurrencyCode                string
	PaymentTermCode             *int64
	AdditionalDays              int
	BearerCode                  *int64
	SaleDate                    *time.Time
	TotalWeightNet              float64
	TotalWeightGross            float64
	TotalGross                  float64
	TotalNet                    float64
	TotalNetNoST                float64
	TotalWithIPIWithST          float64
	Notes                       *string
	ObsCustomer                 *string
	IsBlocked                   bool
	BlockReason                 *string
	IsFirm                      bool
	IsActive                    bool
	IsNFCe                      bool
	Street                      *string
	StreetNumber                *string
	ForeignDocument             *string
	CollectionEstablishmentCode *int64
	NFTypeDescription           *string
	CarrierCode                 *int64
	FreightType                 *string
	FreightValue                float64
	InsuranceValue              float64
	VolumeQuantity              float64
	VolumeType                  *string
	NetWeight                   float64
	GrossWeight                 float64
	DiscountValue               float64
	SurchargeValue              float64
	ProjectCode                 *string
	ProjectName                 *string
	Items                       []*SalesOrderItem
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
	CreatedBy                   uuid.UUID
}

type SalesOrderItem struct {
	Code             int64
	SalesOrderCode   int64
	Sequence         int
	ItemCode         int64
	Mask             string
	DigitDate        time.Time
	NFType           *string
	SalesUOM         *string
	WarehouseCode    *int64
	PriceTableCode   *int64
	RequestedQty     float64
	UnitPrice        float64
	AttendedQty      float64
	CancelledQty     float64
	Balance          float64 // derived: RequestedQty - AttendedQty - CancelledQty
	DeliveryDate     *time.Time
	DeliveryDateFirm bool
	CustomerDelivery *string
	Lot              *string
	CouponDelivery   *string
	PaidAtCashier    bool
	IPIPct           float64
	ICMSPct          float64
	PISPct           float64
	COFINSPct        float64
	STPct            float64
	DiscountPct      float64
	TotalGross       float64
	TotalNet         float64
	TotalNetWithIPI  float64
	TotalIPI         float64
	TotalST          float64
	UnitWeightNet    float64
	UnitWeightGross  float64
	Status           SalesOrderItemStatus
	Notes            *string
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
