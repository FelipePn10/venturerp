package response

import (
	"time"

	"github.com/google/uuid"
)

// EmployeeResponse is the API representation of an employee.
type EmployeeResponse struct {
	ID                 int64     `json:"id"`
	Code               int64     `json:"code"`
	Name               string    `json:"name"`
	Situation          string    `json:"situation"`
	ParticipatesBudget bool      `json:"participates_budget"`
	TechnicalAssistant bool      `json:"technical_assistant"`
	Role               string    `json:"role"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	CreatedBy          uuid.UUID `json:"created_by"`
}
