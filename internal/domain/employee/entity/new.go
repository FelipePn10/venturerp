package entity

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

func NewEmployee(code int64, name, role string, participatesBudget, technicalAssistant bool, createdBy uuid.UUID) (*Employee, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("employee name is required")
	}
	if code <= 0 {
		return nil, errors.New("employee code must be positive")
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
