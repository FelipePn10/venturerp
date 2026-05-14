package entity

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

func NewProductionPlan(code int64, name, independentDemands string, groupSameDateOrders bool, planningTypes []string, createdBy uuid.UUID) (*ProductionPlan, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("production plan name is required")
	}
	if code <= 0 {
		return nil, errors.New("production plan code must be positive")
	}
	switch independentDemands {
	case IndependentDemandsNo, IndependentDemandsFromDate, IndependentDemandsAll:
	default:
		return nil, errors.New("invalid independent_demands value: must be NO, FROM_DATE or ALL")
	}
	if len(planningTypes) == 0 {
		planningTypes = []string{"MRP"}
	}
	return &ProductionPlan{
		Code:                code,
		Name:                name,
		IndependentDemands:  independentDemands,
		GroupSameDateOrders: groupSameDateOrders,
		PlanningTypes:       planningTypes,
		Parameters:          map[string]interface{}{},
		IsActive:            true,
		CreatedBy:           createdBy,
	}, nil
}
