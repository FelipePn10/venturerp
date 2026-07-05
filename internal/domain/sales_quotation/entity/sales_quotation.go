package entity

import (
	"time"

	"github.com/google/uuid"
)

type SalesQuotationStatus string

const (
	SalesQuotationStatusDraft          SalesQuotationStatus = "R"
	SalesQuotationStatusWebOrder       SalesQuotationStatus = "P"
	SalesQuotationStatusAnalysis       SalesQuotationStatus = "A"
	SalesQuotationStatusBudgetAnalysis SalesQuotationStatus = "OA"
	SalesQuotationStatusERPOrder       SalesQuotationStatus = "F"
	SalesQuotationStatusERPBudget      SalesQuotationStatus = "OF"
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
	ProbabilityPct          float64
	CommissionPct           float64
	IsNFCe                  bool
	Street                  *string
	StreetNumber            *string
	ForeignDocument         *string
	ReleaseStatus           SalesQuotationReleaseStatus
	CommercialBlocked       bool
	CommercialBlockReason   *string
	CarrierCode             *int64
	FreightType             *string
	VerifyFreight           bool
	FreightValue            float64
	RedeliveryFreightValue  float64
	InsuranceValue          float64
	DiscountValue           float64
	SurchargeValue          float64
	RetainedTaxValue        float64
	TotalGross              float64
	TotalNet                float64
	DeliveryAuthorization   *string
	Notes                   *string
	ObsCustomer             *string
	CancelReason            *string
	CancelComplement        *string
	AttendedReason          *string
	AttendedAt              *time.Time
	ConvertedSalesOrderCode *int64
	ConvertedAt             *time.Time
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
	RequestedQty       float64
	UnitPrice          float64
	AttendedQty        float64
	CancelledQty       float64
	Balance            float64
	DeliveryDate       *time.Time
	DeliveryDateFirm   bool
	DiscountPct        float64
	IPIPct             float64
	STPct              float64
	TotalGross         float64
	TotalNet           float64
	TotalNetWithIPI    float64
	Status             SalesQuotationItemStatus
	Notes              *string
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}
