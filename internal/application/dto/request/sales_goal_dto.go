package request

type CreateSalesGoalPeriodDTO struct {
	Description string `json:"description"`
	PeriodType  string `json:"period_type"`
	StartDate   string `json:"start_date"`
	EndDate     string `json:"end_date"`
}

type CreateSalesGoalDTO struct {
	RepresentativeCode int64   `json:"representative_code"`
	PeriodCode         int64   `json:"period_code"`
	AnalysisBase       string  `json:"analysis_base"`
	AwardPct           float64 `json:"award_pct"`
	Notes              *string `json:"notes,omitempty"`
}

type UpdateSalesGoalDTO struct {
	Code               int64   `json:"code"`
	RepresentativeCode int64   `json:"representative_code"`
	PeriodCode         int64   `json:"period_code"`
	AnalysisBase       string  `json:"analysis_base"`
	AwardPct           float64 `json:"award_pct"`
	Notes              *string `json:"notes,omitempty"`
	IsActive           bool    `json:"is_active"`
}

type SalesGoalItemDTO struct {
	GoalCode               int64   `json:"goal_code"`
	TargetType             string  `json:"target_type"`
	ItemCode               *int64  `json:"item_code,omitempty"`
	ItemClassificationCode *int64  `json:"item_classification_code,omitempty"`
	ItemGroupCode          *int64  `json:"item_group_code,omitempty"`
	SalesUOM               *string `json:"sales_uom,omitempty"`
	TargetQuantity         float64 `json:"target_quantity"`
	TargetValue            float64 `json:"target_value"`
	BonusPct               float64 `json:"bonus_pct"`
	IsActive               bool    `json:"is_active"`
}

type SalesGoalGroupTargetDTO struct {
	PeriodCode          int64   `json:"period_code"`
	CommercialGroupCode int64   `json:"commercial_group_code"`
	GoalType            string  `json:"goal_type"`
	MinimumValue        float64 `json:"minimum_value"`
	MinimumBonusPct     float64 `json:"minimum_bonus_pct"`
	ProbableValue       float64 `json:"probable_value"`
	ProbableBonusPct    float64 `json:"probable_bonus_pct"`
	IdealValue          float64 `json:"ideal_value"`
	IdealBonusPct       float64 `json:"ideal_bonus_pct"`
	IsActive            bool    `json:"is_active"`
}

type SalesGoalGroupCustomerDTO struct {
	GroupGoalID        int64   `json:"group_goal_id"`
	CustomerCode       int64   `json:"customer_code"`
	RepresentativeCode *int64  `json:"representative_code,omitempty"`
	MinimumValue       float64 `json:"minimum_value"`
	MinimumBonusPct    float64 `json:"minimum_bonus_pct"`
	ProbableValue      float64 `json:"probable_value"`
	ProbableBonusPct   float64 `json:"probable_bonus_pct"`
	IdealValue         float64 `json:"ideal_value"`
	IdealBonusPct      float64 `json:"ideal_bonus_pct"`
	IsActive           bool    `json:"is_active"`
}

type SalesGoalBalanceDTO struct {
	PeriodCode          int64   `json:"period_code"`
	NextPeriodCode      *int64  `json:"next_period_code,omitempty"`
	BalanceScope        string  `json:"balance_scope"`
	RepresentativeCode  *int64  `json:"representative_code,omitempty"`
	CommercialGroupCode *int64  `json:"commercial_group_code,omitempty"`
	CustomerCode        *int64  `json:"customer_code,omitempty"`
	GoalType            string  `json:"goal_type"`
	RealizedValue       float64 `json:"realized_value"`
	IdealValue          float64 `json:"ideal_value"`
	BalanceValue        float64 `json:"balance_value"`
	Notes               *string `json:"notes,omitempty"`
}
