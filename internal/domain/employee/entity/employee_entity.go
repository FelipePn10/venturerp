package entity

import (
	"time"

	"github.com/google/uuid"
)

type EmployeeSituation string

const (
	EmployeeActive   EmployeeSituation = "ACTIVE"
	EmployeeInactive EmployeeSituation = "INACTIVE"
)

type Employee struct {
	ID                 int64
	Code               int64
	Name               string
	Situation          EmployeeSituation
	ParticipatesBudget bool
	TechnicalAssistant bool
	Role               string // PLANNER, OPERATOR, MANAGER, etc.
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CreatedBy          uuid.UUID
}
