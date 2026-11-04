package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

type Period struct {
	Code        int64
	Description string
	PeriodType  string
	StartDate   time.Time
	EndDate     time.Time
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Goal struct {
	Code               int64
	RepresentativeCode int64
	PeriodCode         int64
	AnalysisBase       string
	AwardPct           decimal.Decimal
	Notes              *string
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Items              []*GoalItem
}

type GoalItem struct {
	ID                     int64
	GoalCode               int64
	TargetType             string
	ItemCode               *int64
	ItemClassificationCode *int64
	ItemGroupCode          *int64
	SalesUOM               *string
	TargetQuantity         decimal.Decimal
	TargetValue            decimal.Decimal
	BonusPct               decimal.Decimal
	IsActive               bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type GroupTarget struct {
	ID                  int64
	PeriodCode          int64
	CommercialGroupCode int64
	GoalType            string
	MinimumValue        decimal.Decimal
	MinimumBonusPct     decimal.Decimal
	ProbableValue       decimal.Decimal
	ProbableBonusPct    decimal.Decimal
	IdealValue          decimal.Decimal
	IdealBonusPct       decimal.Decimal
	IsActive            bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
	Customers           []*GroupCustomer
}

type GroupCustomer struct {
	ID                 int64
	GroupGoalID        int64
	CustomerCode       int64
	RepresentativeCode *int64
	MinimumValue       decimal.Decimal
	MinimumBonusPct    decimal.Decimal
	ProbableValue      decimal.Decimal
	ProbableBonusPct   decimal.Decimal
	IdealValue         decimal.Decimal
	IdealBonusPct      decimal.Decimal
	IsActive           bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Balance struct {
	ID                  int64
	PeriodCode          int64
	NextPeriodCode      *int64
	BalanceScope        string
	RepresentativeCode  *int64
	CommercialGroupCode *int64
	CustomerCode        *int64
	GoalType            string
	RealizedValue       decimal.Decimal
	IdealValue          decimal.Decimal
	BalanceValue        decimal.Decimal
	Notes               *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
