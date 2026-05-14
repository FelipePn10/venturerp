package request

import "github.com/google/uuid"

type CreateEmployeeDTO struct {
	Code               int64     `json:"code"`
	Name               string    `json:"name"`
	Role               string    `json:"role"`
	ParticipatesBudget bool      `json:"participates_budget"`
	TechnicalAssistant bool      `json:"technical_assistant"`
	CreatedBy          uuid.UUID `json:"created_by"`
}

type UpdateEmployeeDTO struct {
	Code               int64  `json:"code"`
	Name               string `json:"name"`
	Role               string `json:"role"`
	Situation          string `json:"situation"`
	ParticipatesBudget bool   `json:"participates_budget"`
	TechnicalAssistant bool   `json:"technical_assistant"`
}
