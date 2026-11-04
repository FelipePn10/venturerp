package repository

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_goal/entity"
)

type PeriodFilter struct {
	From       *time.Time
	To         *time.Time
	OnlyActive bool
}

type GoalFilter struct {
	RepresentativeCode *int64
	PeriodCode         *int64
	AnalysisBase       string
	OnlyActive         bool
}

type ReportFilter struct {
	RepresentativeCode *int64
	CustomerCode       *int64
	RegionCode         *int64
	MicroregionCode    *int64
	PeriodCode         *int64
	From               *time.Time
	To                 *time.Time
	AnalysisBase       string
	Layout             string
	BreakBy            string
	IncludeMissedItems bool
}

type ReportRow struct {
	Scope               string
	RepresentativeCode  *int64
	CommercialGroupCode *int64
	CustomerCode        *int64
	PeriodCode          int64
	PeriodDescription   string
	AnalysisBase        string
	TargetValue         decimal.Decimal
	TargetQuantity      decimal.Decimal
	RealizedValue       decimal.Decimal
	RealizedQuantity    decimal.Decimal
	BalanceValue        decimal.Decimal
	AchievementPct      decimal.Decimal
	BonusPct            decimal.Decimal
	Status              string
}

type Repository interface {
	CreatePeriod(ctx context.Context, p *entity.Period) (*entity.Period, error)
	ListPeriods(ctx context.Context, filter PeriodFilter) ([]*entity.Period, error)
	GetPeriod(ctx context.Context, code int64) (*entity.Period, error)

	CreateGoal(ctx context.Context, g *entity.Goal) (*entity.Goal, error)
	UpdateGoal(ctx context.Context, g *entity.Goal) (*entity.Goal, error)
	GetGoal(ctx context.Context, code int64) (*entity.Goal, error)
	ListGoals(ctx context.Context, filter GoalFilter) ([]*entity.Goal, error)
	AddGoalItem(ctx context.Context, item *entity.GoalItem) (*entity.GoalItem, error)

	UpsertGroupTarget(ctx context.Context, target *entity.GroupTarget) (*entity.GroupTarget, error)
	AddGroupCustomer(ctx context.Context, customer *entity.GroupCustomer) (*entity.GroupCustomer, error)
	UpsertBalance(ctx context.Context, balance *entity.Balance) (*entity.Balance, error)

	Report(ctx context.Context, filter ReportFilter) ([]ReportRow, error)
}
