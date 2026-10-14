package entity

import (
	"github.com/FelipePn10/panossoerp/internal/domain/domainerr"
	"strings"

	"github.com/google/uuid"
)

func NewEmployee(code int64, name, role string, participatesBudget, technicalAssistant bool, createdBy uuid.UUID) (*Employee, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domainerr.NewValidation("informe o nome do funcionário")
	}
	if code <= 0 {
		return nil, domainerr.NewValidation("o código do funcionário deve ser maior que zero")
	}
	role = strings.TrimSpace(role)
	if role == "" {
		role = "PLANNER"
	}
	return &Employee{
		Code:               code,
		Name:               name,
		Situation:          EmployeeActive,
		ParticipatesBudget: participatesBudget,
		TechnicalAssistant: technicalAssistant,
		Role:               role,
		CreatedBy:          createdBy,
	}, nil
}
