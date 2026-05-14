package entity

import (
	"time"

	"github.com/google/uuid"
)

type SalesDivisionAnalysis string

const (
	AnalysisFree          SalesDivisionAnalysis = "FREE"
	AnalysisBlockAlways   SalesDivisionAnalysis = "BLOCK_ALWAYS"
	AnalysisAlwaysAnalyze SalesDivisionAnalysis = "ALWAYS_ANALYZE"
)

type SalesDivision struct {
	ID                      int64
	Code                    int64
	Description             string
	CommercialAnalysis      SalesDivisionAnalysis
	FinancialAnalysis       SalesDivisionAnalysis
	IsTechnicalAssistance   bool
	ConsiderDeliveryPromise bool
	ConsiderMRP             bool
	AllowOutsideLimits      bool
	MinimumDeliveryDays     int
	FinancialDelayDays      int
	PISPercentage           float64
	CofinsPercentage        float64
	ParentDivisionID        *int64
	IsActive                bool
	CreatedAt               time.Time
	UpdatedAt               time.Time
	CreatedBy               uuid.UUID
}
