package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SalesQuotationStatus string

const (
	SalesQuotationStatusDraft          SalesQuotationStatus = "R"
	SalesQuotationStatusWebOrder       SalesQuotationStatus = "P"
	SalesQuotationStatusAnalysis       SalesQuotationStatus = "A"
	SalesQuotationStatusBudgetAnalysis SalesQuotationStatus = "OA"
	SalesQuotationStatusERPOrder       SalesQuotationStatus = "F"
	SalesQuotationStatusERPBudget      SalesQuotationStatus = "OF"
	SalesQuotationStatusVentureOrder   SalesQuotationStatus = "V"
	SalesQuotationStatusVentureBudget  SalesQuotationStatus = "OV"
	SalesQuotationStatusCancelled      SalesQuotationStatus = "CANCELLED"
	SalesQuotationStatusAttended       SalesQuotationStatus = "ATTENDED"
	SalesQuotationStatusExpired        SalesQuotationStatus = "EXPIRED"
)

type SalesQuotationType string

const (
	SalesQuotationTypeThirdParty  SalesQuotationType = "API_TERCEIROS"
	SalesQuotationTypeConsult     SalesQuotationType = "CONSULTA"
	SalesQuotationTypePortal      SalesQuotationType = "FOCCOPORTAL"
	SalesQuotationTypeImported    SalesQuotationType = "IMPORTADO"
	SalesQuotationTypeNegotiation SalesQuotationType = "NEGOCIACAO"
	SalesQuotationTypeSale        SalesQuotationType = "VENDA"
)

type SalesQuotationReleaseStatus string

const (
	SalesQuotationReleaseBlocked SalesQuotationReleaseStatus = "BLOCKED"
	SalesQuotationReleaseManual  SalesQuotationReleaseStatus = "MANUAL_RELEASED"
	SalesQuotationReleaseOK      SalesQuotationReleaseStatus = "RELEASED"
)

type SalesQuotationItemStatus string

const (
	SalesQuotationItemStatusOpen      SalesQuotationItemStatus = "OPEN"
	SalesQuotationItemStatusPartial   SalesQuotationItemStatus = "PARTIAL"
	SalesQuotationItemStatusDelivered SalesQuotationItemStatus = "DELIVERED"
	SalesQuotationItemStatusCancelled SalesQuotationItemStatus = "CANCELLED"
)

type SalesQuotation struct {
	Code                    int64
	QuotationNumber         int64
	EnterpriseCode          int64
	Status                  SalesQuotationStatus
	QuotationType           SalesQuotationType
	EmissionDate            time.Time
	DigitDate               time.Time
	ValidUntil              *time.Time
	DeliveryDate            *time.Time
	DeliveryDateFirm        bool
	PurchaseOrderNumber     *string
	CustomerCode            *int64
	BillingAddressCode      *int64
	ShippingAddressCode     *int64
	RepresentativeCode      *int64
	SalesDivisionCode       *int64
	PriceTableCode          *int64
	PaymentTermCode         *int64
	CurrencyCode            string
	ProbabilityPct          decimal.Decimal
	CommissionPct           decimal.Decimal
	IsNFCe                  bool
	DeliveryWithReceipt     bool
	Street                  *string
	StreetNumber            *string
	ForeignDocument         *string
	ReleaseStatus           SalesQuotationReleaseStatus
	CommercialBlocked       bool
	CommercialBlockReason   *string
	CarrierCode             *int64
	FreightType             *string
	VerifyFreight           bool
	FreightValue            decimal.Decimal
	RedeliveryFreightValue  decimal.Decimal
	InsuranceValue          decimal.Decimal
	DiscountValue           decimal.Decimal
	SurchargeValue          decimal.Decimal
	RetainedTaxValue        decimal.Decimal
	TotalGross              decimal.Decimal
	TotalNet                decimal.Decimal
	DeliveryAuthorization   *string
	Notes                   *string
	ObsCustomer             *string
	CancelReason            *string
	CancelComplement        *string
	AttendedReason          *string
	AttendedAt              *time.Time
	ConvertedSalesOrderCode *int64
	ConvertedAt             *time.Time
	DAVGeneratedAt          *time.Time
	DAVReportKey            *uuid.UUID
	ConsumerAddress         *string
	IsActive                bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
	CreatedBy               uuid.UUID
	Items                   []*SalesQuotationItem
}

type SalesQuotationItem struct {
	Code               int64
	SalesQuotationCode int64
	Sequence           int
	ItemCode           int64
	Mask               string
	SalesUOM           *string
	WarehouseCode      *int64
	PriceTableCode     *int64
	RequestedQty       decimal.Decimal
	UnitPrice          decimal.Decimal
	AttendedQty        decimal.Decimal
	CancelledQty       decimal.Decimal
	Balance            decimal.Decimal
	DeliveryDate       *time.Time
	DeliveryDateFirm   bool
	DiscountPct        decimal.Decimal
	IPIPct             decimal.Decimal
	STPct              decimal.Decimal
	TotalGross         decimal.Decimal
	TotalNet           decimal.Decimal
	TotalNetWithIPI    decimal.Decimal
	Status             SalesQuotationItemStatus
	Notes              *string
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
