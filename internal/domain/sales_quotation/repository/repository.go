package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	"github.com/shopspring/decimal"
)

type SalesQuotationFilter struct {
	QuotationNumber     *int64
	CustomerCode        *int64
	SalesDivisionCode   *int64
	QuotationType       *entity.SalesQuotationType
	Status              *entity.SalesQuotationStatus
	From                *time.Time
	To                  *time.Time
	PurchaseOrderNumber *string
	FreightType         *string
	Limit               int
	Offset              int
}

type SalesQuotationReport struct {
	TotalQuotations int64
	TotalGross      decimal.Decimal
	TotalNet        decimal.Decimal
	OpenCount       int64
	ApprovedCount   int64
	ConvertedCount  int64
	CancelledCount  int64
	ExpiredCount    int64
	WeightedNet     decimal.Decimal
	RetainedTax     decimal.Decimal
}

type SalesQuotationRepository interface {
	NextQuotationNumber(ctx context.Context, enterpriseCode int64) (int64, error)
	Create(ctx context.Context, quotation *entity.SalesQuotation) (*entity.SalesQuotation, error)
	Update(ctx context.Context, quotation *entity.SalesQuotation) (*entity.SalesQuotation, error)
	GetByCode(ctx context.Context, code int64) (*entity.SalesQuotation, error)
	List(ctx context.Context, filter SalesQuotationFilter) ([]*entity.SalesQuotation, error)
	Cancel(ctx context.Context, code, reasonCode int64, reason string, complement *string) error
	Uncancel(ctx context.Context, code, reasonCode int64, reason string, complement *string) error
	Attend(ctx context.Context, code int64, reason string, complement *string, eventDate time.Time) error
	ChangeStatus(ctx context.Context, code int64, status entity.SalesQuotationStatus) error
	ChangeRelease(ctx context.Context, code int64, status entity.SalesQuotationReleaseStatus, reason string) error
	ListEvents(ctx context.Context, code int64) ([]*entity.Event, error)
	MarkConverted(ctx context.Context, quotationCode, salesOrderCode int64) error
	Report(ctx context.Context, filter SalesQuotationFilter) (*SalesQuotationReport, error)
	GetParameters(ctx context.Context) (*entity.Parameters, error)
	SaveParameters(ctx context.Context, parameters *entity.Parameters) (*entity.Parameters, error)
	SaveCommissionPattern(ctx context.Context, pattern *entity.CommissionPattern) (*entity.CommissionPattern, error)
	ListCommissionPatterns(ctx context.Context) ([]*entity.CommissionPattern, error)
	SaveCancellationReason(ctx context.Context, reason *entity.CancellationReason) (*entity.CancellationReason, error)
	ListCancellationReasons(ctx context.Context) ([]*entity.CancellationReason, error)
	GetCancellationReason(ctx context.Context, code int64) (*entity.CancellationReason, error)
	GenerateDAV(ctx context.Context, code int64) (*entity.SalesQuotation, error)
	CreateAttachment(ctx context.Context, attachment *entity.Attachment) (*entity.Attachment, error)
	ListAttachments(ctx context.Context, quotationCode int64) ([]*entity.Attachment, error)
	GetAttachment(ctx context.Context, quotationCode, attachmentID int64) (*entity.Attachment, error)
	DeleteAttachment(ctx context.Context, quotationCode, attachmentID int64) error

	CreateItem(ctx context.Context, item *entity.SalesQuotationItem) (*entity.SalesQuotationItem, error)
	UpdateItem(ctx context.Context, item *entity.SalesQuotationItem) (*entity.SalesQuotationItem, error)
	GetItem(ctx context.Context, itemCode int64) (*entity.SalesQuotationItem, error)
	ListItems(ctx context.Context, quotationCode int64) ([]*entity.SalesQuotationItem, error)
	CancelItem(ctx context.Context, itemCode, reasonCode int64, reason string, complement *string) error
	RecalculateTotals(ctx context.Context, quotationCode int64) error
}
