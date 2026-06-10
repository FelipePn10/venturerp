package response

import (
	"time"

	"github.com/google/uuid"
)

// ProductionPlanResponse is the API representation of a production plan.
type ProductionPlanResponse struct {
	ID                  int64                  `json:"id"`
	Code                int64                  `json:"code"`
	Name                string                 `json:"name"`
	IndependentDemands  string                 `json:"independent_demands"`
	GroupSameDateOrders bool                   `json:"group_same_date_orders"`
	PlanningTypes       []string               `json:"planning_types"`
	Classification      *string                `json:"classification,omitempty"`
	ClassItemCodes      *string                `json:"class_item_codes,omitempty"`
	OrderItemCode       *int64                 `json:"order_item_code,omitempty"`
	LastCalculatedAt    *time.Time             `json:"last_calculated_at,omitempty"`
	Parameters          map[string]interface{} `json:"parameters,omitempty"`
	IsActive            bool                   `json:"is_active"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	CreatedBy           uuid.UUID              `json:"created_by"`
}
