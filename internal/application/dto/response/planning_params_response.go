package response

import (
	"time"

	"github.com/google/uuid"
)

// PlanningParamResponse is the API representation of a planning parameter.
type PlanningParamResponse struct {
	ID          int64     `json:"id"`
	ParamNumber int       `json:"param_number"`
	ParamKey    string    `json:"param_key"`
	Value       string    `json:"value"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	UpdatedBy   uuid.UUID `json:"updated_by"`
}
