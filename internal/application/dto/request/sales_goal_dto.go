package request

import "github.com/shopspring/decimal"

type CreateSalesGoalPeriodDTO struct {
	Description string `json:"description"`
	PeriodType  string `json:"period_type"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
}

type CreateSalesGoalDTO struct {
	RepresentativeCode int64           `json:"representative_code"`
	PeriodCode         int64           `json:"period_code"`
	AnalysisBase       string          `json:"analysis_base"`
	AwardPct           decimal.Decimal `json:"award_pct"`
	Notes              *string         `json:"notes,omitempty"`
}

type UpdateSalesGoalDTO struct {
	Code               int64           `json:"code"`
	RepresentativeCode int64           `json:"representative_code"`
	PeriodCode         int64           `json:"period_code"`
	AnalysisBase       string          `json:"analysis_base"`
	AwardPct           decimal.Decimal `json:"award_pct"`
	Notes              *string         `json:"notes,omitempty"`
	IsActive           *bool           `json:"is_active,omitempty"`
}

type SalesGoalItemDTO struct {
	GoalCode               int64           `json:"goal_code"`
	TargetType             string          `json:"target_type"`
	ItemCode               *int64          `json:"item_code,omitempty"`
	ItemClassificationCode *int64          `json:"item_classification_code,omitempty"`
	ItemGroupCode          *int64          `json:"item_group_code,omitempty"`
	SalesUOM               *string         `json:"sales_uom,omitempty"`
	TargetQuantity         decimal.Decimal `json:"target_quantity"`
	TargetValue            decimal.Decimal `json:"target_value"`
	BonusPct               decimal.Decimal `json:"bonus_pct"`
	IsActive               bool            `json:"is_active"`
}

type SalesGoalGroupTargetDTO struct {
	PeriodCode          int64           `json:"period_code"`
	CommercialGroupCode int64           `json:"commercial_group_code"`
	GoalType            string          `json:"goal_type"`
	MinimumValue        decimal.Decimal `json:"minimum_value"`
	MinimumBonusPct     decimal.Decimal `json:"minimum_bonus_pct"`
	ProbableValue       decimal.Decimal `json:"probable_value"`
	ProbableBonusPct    decimal.Decimal `json:"probable_bonus_pct"`
	IdealValue          decimal.Decimal `json:"ideal_value"`
	IdealBonusPct       decimal.Decimal `json:"ideal_bonus_pct"`
	IsActive            bool            `json:"is_active"`
}

type SalesGoalGroupCustomerDTO struct {
	GroupGoalID        int64           `json:"group_goal_id"`
	CustomerCode       int64           `json:"customer_code"`
	RepresentativeCode *int64          `json:"representative_code,omitempty"`
	MinimumValue       decimal.Decimal `json:"minimum_value"`
	MinimumBonusPct    decimal.Decimal `json:"minimum_bonus_pct"`
	ProbableValue      decimal.Decimal `json:"probable_value"`
	ProbableBonusPct   decimal.Decimal `json:"probable_bonus_pct"`
	IdealValue         decimal.Decimal `json:"ideal_value"`
	IdealBonusPct      decimal.Decimal `json:"ideal_bonus_pct"`
	IsActive           bool            `json:"is_active"`
}

type SalesGoalBalanceDTO struct {
	PeriodCode          int64           `json:"period_code"`
	NextPeriodCode      *int64          `json:"next_period_code,omitempty"`
	BalanceScope        string          `json:"balance_scope"`
	RepresentativeCode  *int64          `json:"representative_code,omitempty"`
	CommercialGroupCode *int64          `json:"commercial_group_code,omitempty"`
	CustomerCode        *int64          `json:"customer_code,omitempty"`
	GoalType            string          `json:"goal_type"`
	RealizedValue       decimal.Decimal `json:"realized_value"`
	IdealValue          decimal.Decimal `json:"ideal_value"`
	BalanceValue        decimal.Decimal `json:"balance_value"`
	Notes               *string         `json:"notes,omitempty"`
}
