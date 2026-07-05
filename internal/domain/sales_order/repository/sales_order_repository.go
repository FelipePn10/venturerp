package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/google/uuid"
)

type SalesOrderFilter struct {
	CustomerCode             *int64
	RepresentativeCode       *int64
	PaymentTermCode          *int64
	Status                   *entity.SalesOrderStatus
	CommercialAnalysisStatus *entity.SalesOrderAnalysisStatus
	FinancialAnalysisStatus  *entity.SalesOrderAnalysisStatus
	ReleaseStatus            *entity.SalesOrderReleaseStatus
	ConferenceStatus         *entity.SalesOrderConferenceStatus
	IsBlocked                *bool
	EmissionFrom             *time.Time
	EmissionTo               *time.Time
	DeliveryFrom             *time.Time
	DeliveryTo               *time.Time
}

type SalesOrderReport struct {
	TotalOrders            int64
	TotalGross             float64
	TotalNet               float64
	OpenCount              int64
	ConfirmedCount         int64
	InvoicedCount          int64
	CancelledCount         int64
	BlockedCount           int64
	CommercialPendingCount int64
	FinancialPendingCount  int64
	ConferencePendingCount int64
	DelayedCount           int64
}

type SalesOrderRepository interface {
	NextOrderNumber(ctx context.Context, enterpriseCode int64) (int64, error)
	Create(ctx context.Context, o *entity.SalesOrder) (*entity.SalesOrder, error)
	Update(ctx context.Context, o *entity.SalesOrder) (*entity.SalesOrder, error)
	GetByCode(ctx context.Context, code int64) (*entity.SalesOrder, error)
	List(ctx context.Context) ([]*entity.SalesOrder, error)
	ListByCustomer(ctx context.Context, customerCode int64) ([]*entity.SalesOrder, error)
	ListByStatus(ctx context.Context, status entity.SalesOrderStatus) ([]*entity.SalesOrder, error)
	ListByDateRange(ctx context.Context, from, to time.Time) ([]*entity.SalesOrder, error)
	ListAdvanced(ctx context.Context, filter SalesOrderFilter) ([]*entity.SalesOrder, error)
	Report(ctx context.Context, filter SalesOrderFilter) (*SalesOrderReport, error)
	Cancel(ctx context.Context, code int64, reason string, complement *string) error
	Block(ctx context.Context, code int64, reason string) error
	Unblock(ctx context.Context, code int64) error
	ChangeStatus(ctx context.Context, code int64, status entity.SalesOrderStatus) error
	Analyze(ctx context.Context, code int64, area string, status entity.SalesOrderAnalysisStatus, reason string, createdBy uuid.UUID) error
	Release(ctx context.Context, code int64, releaseStatus entity.SalesOrderReleaseStatus, reason string, area string, createdBy uuid.UUID) error
	Attend(ctx context.Context, code int64, reason string, eventDate *time.Time, createdBy uuid.UUID) error
	Confer(ctx context.Context, code int64, status entity.SalesOrderConferenceStatus, reason string, createdBy uuid.UUID) error
	SaveDelayReason(ctx context.Context, code int64, reason, action string, createdBy uuid.UUID) error

	CreateItem(ctx context.Context, item *entity.SalesOrderItem) (*entity.SalesOrderItem, error)
	UpdateItem(ctx context.Context, item *entity.SalesOrderItem) (*entity.SalesOrderItem, error)
	ListItems(ctx context.Context, salesOrderCode int64) ([]*entity.SalesOrderItem, error)
	CancelItem(ctx context.Context, itemCode int64) error
}
