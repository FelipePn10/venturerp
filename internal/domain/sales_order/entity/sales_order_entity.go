package entity

import (
	"github.com/shopspring/decimal"
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
	SalesOrderOriginNormal       SalesOrderOrigin = "NORMAL"
	SalesOrderOriginDependent    SalesOrderOrigin = "DEPENDENT"
	SalesOrderOriginAssistance   SalesOrderOrigin = "ASSISTANCE"
	SalesOrderOriginReserve      SalesOrderOrigin = "RESERVE"
	SalesOrderOriginCopy         SalesOrderOrigin = "COPY"
	SalesOrderOriginInterFactory SalesOrderOrigin = "INTER_FACTORY"
)

// SalesOrderItemStatus represents the fulfillment status of an order line.
type SalesOrderItemStatus string

const (
	SalesOrderItemStatusOpen      SalesOrderItemStatus = "OPEN"
	SalesOrderItemStatusPartial   SalesOrderItemStatus = "PARTIAL"
	SalesOrderItemStatusDelivered SalesOrderItemStatus = "DELIVERED"
	SalesOrderItemStatusCancelled SalesOrderItemStatus = "CANCELLED"
)

type SalesOrderAnalysisStatus string

const (
	SalesOrderAnalysisNotAnalyzed SalesOrderAnalysisStatus = "NOT_ANALYZED"
	SalesOrderAnalysisApproved    SalesOrderAnalysisStatus = "APPROVED"
	SalesOrderAnalysisRejected    SalesOrderAnalysisStatus = "REJECTED"
)

type SalesOrderReleaseStatus string

const (
	SalesOrderReleaseBlocked SalesOrderReleaseStatus = "BLOCKED"
	SalesOrderReleaseManual  SalesOrderReleaseStatus = "MANUAL_RELEASED"
	SalesOrderReleaseOK      SalesOrderReleaseStatus = "RELEASED"
)

type SalesOrderConferenceStatus string

const (
	SalesOrderConferencePending   SalesOrderConferenceStatus = "PENDING"
	SalesOrderConferenceConferred SalesOrderConferenceStatus = "CONFERRED"
	SalesOrderConferenceDivergent SalesOrderConferenceStatus = "DIVERGENT"
)

type SalesOrder struct {
	Code                      int64
	OrderNumber               int64
	EnterpriseCode            int64
	Status                    SalesOrderStatus
	Origin                    SalesOrderOrigin
	EmissionDate              time.Time
	DeliveryDate              *time.Time
	DeliveryDateFirm          bool
	DigitDate                 time.Time
	CustomerCode              *int64
	BillingAddressCode        *int64
	ShippingAddressCode       *int64
	RepresentativeCode        *int64
	RepresentativeOrderNumber *int64
	PlanCode                  *int64
	SalesDivisionCode         *int64
	CommissionPct             float64
	TaxTypeCode               *int64
	// FinalidadeCompra do cliente nesta venda (REVENDA, INDUSTRIALIZACAO,
	// USO_CONSUMO); nil = a do cadastro do cliente.
	FinalidadeCompra            *string
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
	TotalGross                  decimal.Decimal
	TotalNet                    decimal.Decimal
	TotalNetNoST                decimal.Decimal
	TotalWithIPIWithST          decimal.Decimal
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
	FreightValue                decimal.Decimal
	InsuranceValue              decimal.Decimal
	VolumeQuantity              float64
	VolumeType                  *string
	NetWeight                   float64
	GrossWeight                 float64
	DiscountValue               decimal.Decimal
	SurchargeValue              decimal.Decimal
	ProjectCode                 *string
	ProjectName                 *string
	CommercialAnalysisStatus    SalesOrderAnalysisStatus
	FinancialAnalysisStatus     SalesOrderAnalysisStatus
	ReleaseStatus               SalesOrderReleaseStatus
	ConferenceStatus            SalesOrderConferenceStatus
	CancelReason                *string
	CancelComplement            *string
	AttendedReason              *string
	AttendedAt                  *time.Time
	DelayReason                 *string
	DelayAction                 *string
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
	RequestedQty     decimal.Decimal
	UnitPrice        decimal.Decimal
	AttendedQty      decimal.Decimal
	CancelledQty     decimal.Decimal
	Balance          decimal.Decimal // derived: RequestedQty - AttendedQty - CancelledQty
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
	TotalGross       decimal.Decimal
	TotalNet         decimal.Decimal
	TotalNetWithIPI  decimal.Decimal
	TotalIPI         decimal.Decimal
	TotalST          decimal.Decimal
	UnitWeightNet    float64
	UnitWeightGross  float64
	Status           SalesOrderItemStatus
	Notes            *string
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// RecalcularTotais refaz os totais da linha em decimal: preço com desconto ×
// quantidade, IPI e ST sobre o bruto, e o saldo. Quatro casas, a escala da
// coluna — o mesmo valor que o float gravava, sem o erro de arredondamento.
func (it *SalesOrderItem) RecalcularTotais() {
	cem := decimal.NewFromInt(100)
	unitario := it.UnitPrice.Mul(cem.Sub(decimal.NewFromFloat(it.DiscountPct))).Div(cem)
	it.TotalGross = unitario.Mul(it.RequestedQty).Round(4)
	ipi := it.TotalGross.Mul(decimal.NewFromFloat(it.IPIPct)).Div(cem).Round(4)
	it.TotalIPI = ipi
	it.TotalNet = it.TotalGross
	it.TotalNetWithIPI = it.TotalGross.Add(ipi)
	it.TotalST = it.TotalGross.Mul(decimal.NewFromFloat(it.STPct)).Div(cem).Round(4)
	it.Balance = it.RequestedQty.Sub(it.AttendedQty).Sub(it.CancelledQty)
}
