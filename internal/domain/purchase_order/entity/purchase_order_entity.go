package entity

import (
	"github.com/shopspring/decimal"
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

	TotalGross    decimal.Decimal
	TotalNet      decimal.Decimal
	TotalDiscount decimal.Decimal

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
	FreightValue     decimal.Decimal
	CarrierCode      *int64
	// Redespacho
	RedispatchCarrierCode  *int64
	RedispatchFreightType  *string
	RedispatchFreightValue decimal.Decimal
	// Adiantamento
	AdvanceDate  *time.Time
	AdvanceValue decimal.Decimal
	// Importação
	IncotermCode *string
	ShipmentDate *time.Time
	// Outros
	TalaoNumber  *string
	AlcadaStatus string // A/B/R/I/N

	IsActive            bool
	IsFirm              bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
	CreatedBy           uuid.UUID
	ResponsibleUserName string

	Items []*PurchaseOrderItem
}

type PurchaseOrderItem struct {
	Code              int64
	PurchaseOrderCode int64
	Sequence          int
	ItemCode          int64
	Mask              string
	RequestedQty      decimal.Decimal
	ReceivedQty       decimal.Decimal
	CancelledQty      decimal.Decimal
	// InvoicedQty é o que já chegou com nota (e virou título a pagar).
	InvoicedQty  decimal.Decimal
	UnitPrice    decimal.Decimal
	TotalPrice   decimal.Decimal
	DiscountPct  decimal.Decimal
	IPIPct       decimal.Decimal
	ICMSPct      decimal.Decimal
	ICMSSTPct    decimal.Decimal
	Status       PurchaseOrderItemStatus
	DeliveryDate *time.Time
	PromisedDate *time.Time
	Notes        *string
	// UM / conversão (compra ↔ estoque)
	PurchaseUOM   *string
	InternalUOM   *string
	InternalQty   decimal.Decimal
	InternalPrice decimal.Decimal
	WarehouseID   *int64
	// Tolerância / cancelamento
	TolerancePct          decimal.Decimal
	CancelledToleranceQty decimal.Decimal
	// Fiscal / contábil
	OperationTypeCode        *int64
	InvoiceTypeCode          *int64
	AccountingAccount        *string
	CostCenterCode           *int64
	FiscalClassificationCode *int64
	// Referências
	RequesterEmployeeCode     *int64
	ContractCode              *int64
	QuotationCode             *int64
	PlannedOrderCode          *int64
	DemandType                *string
	DemandCode                *int64
	SalesOrderCode            *int64
	ProductionOrderID         *int64
	PurchaseRequisitionCode   *int64
	PurchaseRequisitionItemID *int64
	UtilizationType           *string // INDUSTRIALIZACAO | CONSUMO | IMOBILIZADO

	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
