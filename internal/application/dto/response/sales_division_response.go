package response

import (
	"time"

	"github.com/google/uuid"
)

// SalesDivisionResponse is the API representation of a sales division.
type SalesDivisionResponse struct {
	ID                      int64     `json:"id"`
	Code                    int64     `json:"code"`
	Description             string    `json:"description"`
	CommercialAnalysis      string    `json:"commercial_analysis"`
	FinancialAnalysis       string    `json:"financial_analysis"`
	IsTechnicalAssistance   bool      `json:"is_technical_assistance"`
	ConsiderDeliveryPromise bool      `json:"consider_delivery_promise"`
	ConsiderMRP             bool      `json:"consider_mrp"`
	AllowOutsideLimits      bool      `json:"allow_outside_limits"`
	MinimumDeliveryDays     int       `json:"minimum_delivery_days"`
	FinancialDelayDays      int       `json:"financial_delay_days"`
	PISPercentage           float64   `json:"pis_percentage"`
	CofinsPercentage        float64   `json:"cofins_percentage"`
	ParentDivisionID        *int64    `json:"parent_division_id,omitempty"`
	IsActive                bool      `json:"is_active"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
	CreatedBy               uuid.UUID `json:"created_by"`
}
