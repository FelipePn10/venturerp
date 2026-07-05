package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
)

type SalesQuotationFilter struct {
	CustomerCode        *int64
	Status              *entity.SalesQuotationStatus
	From                *time.Time
	To                  *time.Time
	PurchaseOrderNumber *string
	FreightType         *string
}

type SalesQuotationReport struct {
	TotalQuotations int64
	TotalGross      float64
	TotalNet        float64
	OpenCount       int64
	ApprovedCount   int64
	ConvertedCount  int64
	CancelledCount  int64
	ExpiredCount    int64
	WeightedNet     float64
	RetainedTax     float64
}

type SalesQuotationRepository interface {
	NextQuotationNumber(ctx context.Context, enterpriseCode int64) (int64, error)
	Create(ctx context.Context, quotation *entity.SalesQuotation) (*entity.SalesQuotation, error)
	Update(ctx context.Context, quotation *entity.SalesQuotation) (*entity.SalesQuotation, error)
	GetByCode(ctx context.Context, code int64) (*entity.SalesQuotation, error)
	List(ctx context.Context, filter SalesQuotationFilter) ([]*entity.SalesQuotation, error)
	Cancel(ctx context.Context, code int64, reason string, complement *string) error
	Uncancel(ctx context.Context, code int64, reason string, complement *string) error
	Attend(ctx context.Context, code int64, reason string, complement *string, eventDate time.Time) error
	ChangeStatus(ctx context.Context, code int64, status entity.SalesQuotationStatus) error
	MarkConverted(ctx context.Context, quotationCode, salesOrderCode int64) error
	Report(ctx context.Context, filter SalesQuotationFilter) (*SalesQuotationReport, error)

	CreateItem(ctx context.Context, item *entity.SalesQuotationItem) (*entity.SalesQuotationItem, error)
	UpdateItem(ctx context.Context, item *entity.SalesQuotationItem) (*entity.SalesQuotationItem, error)
	ListItems(ctx context.Context, quotationCode int64) ([]*entity.SalesQuotationItem, error)
	CancelItem(ctx context.Context, itemCode int64) error
	RecalculateTotals(ctx context.Context, quotationCode int64) error
}
