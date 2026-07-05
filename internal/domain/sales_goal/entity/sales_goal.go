package entity

import "time"

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
	AwardPct           float64
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
	TargetQuantity         float64
	TargetValue            float64
	BonusPct               float64
	IsActive               bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type GroupTarget struct {
	ID                  int64
	PeriodCode          int64
	CommercialGroupCode int64
	GoalType            string
	MinimumValue        float64
	MinimumBonusPct     float64
	ProbableValue       float64
	ProbableBonusPct    float64
	IdealValue          float64
	IdealBonusPct       float64
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
	MinimumValue       float64
	MinimumBonusPct    float64
	ProbableValue      float64
	ProbableBonusPct   float64
	IdealValue         float64
	IdealBonusPct      float64
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
	RealizedValue       float64
	IdealValue          float64
	BalanceValue        float64
	Notes               *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
