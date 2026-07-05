package repository

import (
	"context"
	"time"

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
	TargetValue         float64
	TargetQuantity      float64
	RealizedValue       float64
	RealizedQuantity    float64
	BalanceValue        float64
	AchievementPct      float64
	BonusPct            float64
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
