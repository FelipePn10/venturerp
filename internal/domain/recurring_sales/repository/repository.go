package repository

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity"
)

type Filter struct {
	EnterpriseCode     *int64
	CustomerCode       *int64
	EstablishmentCode  *int64
	ItemCode           *int64
	ItemClassification *int64
	RepresentativeCode *int64
	MovementType       *entity.MovementType
	OnlyActive         bool
}

type ProjectionFilter struct {
	From               time.Time
	To                 time.Time
	EnterpriseCode     *int64
	CustomerCode       *int64
	ItemCode           *int64
	ItemClassification *int64
	RepresentativeCode *int64
	AdjustmentPercent  float64
}

type Repository interface {
	UpsertParameters(ctx context.Context, p *entity.Parameters) (*entity.Parameters, error)
	GetParameters(ctx context.Context, enterpriseCode int64) (*entity.Parameters, error)
	CreateAdjustmentDate(ctx context.Context, v *entity.AdjustmentDate) (*entity.AdjustmentDate, error)
	ListAdjustmentDates(ctx context.Context, filter Filter) ([]*entity.AdjustmentDate, error)
	Create(ctx context.Context, v *entity.RecurringSale) (*entity.RecurringSale, error)
	Update(ctx context.Context, v *entity.RecurringSale) (*entity.RecurringSale, error)
	Get(ctx context.Context, code int64) (*entity.RecurringSale, error)
	List(ctx context.Context, filter Filter) ([]*entity.RecurringSale, error)
	AddRepresentative(ctx context.Context, v *entity.Representative) (*entity.Representative, error)
	MarkOrderGenerated(ctx context.Context, code int64, orderCode int64) (*entity.RecurringSale, error)
	ClearGeneratedOrder(ctx context.Context, code int64) (*entity.RecurringSale, error)
	Deactivate(ctx context.Context, code int64, reason *string) (*entity.RecurringSale, error)
	CreateAdjustmentLink(ctx context.Context, adjustmentCode, sourceCode int64) error
}
