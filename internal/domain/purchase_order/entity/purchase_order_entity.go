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
	Code                int64
	OrderNumber         int64
	EnterpriseCode      int64
	Status              PurchaseOrderStatus
	Origin              PurchaseOrderOrigin
	EmissionDate        time.Time
	DeliveryDate        *time.Time
	SupplierCode        *int64
	PaymentTermCode     *int64
	CurrencyCode        string
	ShippingAddressCode *int64
	Notes               *string

	TotalGross    float64
	TotalNet      float64
	TotalDiscount float64

	// Comercial / fiscal (capa)
	PriceTableCode   *int64
	InvoiceTypeCode  *int64
	FinancialAccount *string
	RequestTypeCode  *int64
	CurrencyDate     *time.Time
	// Transporte
	FreightType      string  // CIF/DAF/FOB/SEM_FRETE/CONVENIO/RETIRA/CORTESIA/TERCEIROS
	FreightValueType *string // VALOR | PERCENTUAL
	FreightValueMode *string // UNITARIO | TOTAL
	FreightValue     float64
	CarrierCode      *int64
	// Redespacho
	RedispatchCarrierCode  *int64
	RedispatchFreightType  *string
	RedispatchFreightValue float64
	// Adiantamento
	AdvanceDate  *time.Time
	AdvanceValue float64
	// Importação
	IncotermCode *string
	ShipmentDate *time.Time
	// Outros
	TalaoNumber  *string
	AlcadaStatus string // A/B/R/I/N

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
	ICMSSTPct         float64
	Status            PurchaseOrderItemStatus
	DeliveryDate      *time.Time
	PromisedDate      *time.Time
	Notes             *string
	// UM / conversão (compra ↔ estoque)
	PurchaseUOM   *string
	InternalUOM   *string
	InternalQty   float64
	InternalPrice float64
	// Tolerância / cancelamento
	TolerancePct          float64
	CancelledToleranceQty float64
	// Fiscal / contábil
	OperationTypeCode        *int64
	InvoiceTypeCode          *int64
	AccountingAccount        *string
	CostCenterCode           *int64
	FiscalClassificationCode *int64
	// Referências
	RequesterEmployeeCode *int64
	ContractCode          *int64
	QuotationCode         *int64
	UtilizationType       *string // INDUSTRIALIZACAO | CONSUMO | IMOBILIZADO

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
