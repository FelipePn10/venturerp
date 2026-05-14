package request

import "github.com/google/uuid"

type UpdatePlanningParamDTO struct {
	ParamNumber int       `json:"param_number"`
	Value       string    `json:"value"`
	UpdatedBy   uuid.UUID `json:"updated_by"`
}
