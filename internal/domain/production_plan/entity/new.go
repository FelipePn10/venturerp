package entity

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidPlan = errors.New("invalid production plan")
)

var validPlanningTypes = map[string]struct{}{
	"MRP": {}, "MIN_MAX": {}, "REORDER_POINT": {}, "MPS": {}, "KANBAN": {},
}

func NewProductionPlan(code int64, name, independentDemands string, groupSameDateOrders bool, planningTypes []string, createdBy uuid.UUID) (*ProductionPlan, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrInvalidPlan)
	}
	if code <= 0 {
		return nil, fmt.Errorf("%w: code must be positive", ErrInvalidPlan)
	}
	switch independentDemands {
	case IndependentDemandsNo, IndependentDemandsFromDate, IndependentDemandsAll:
	default:
		return nil, fmt.Errorf("%w: independent_demands must be NO, FROM_DATE or ALL", ErrInvalidPlan)
	}
	planningTypes, err := normalizePlanningTypes(planningTypes)
	if err != nil {
		return nil, err
	}
	if createdBy == uuid.Nil {
		return nil, fmt.Errorf("%w: authenticated creator is required", ErrInvalidPlan)
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

func (p *ProductionPlan) Configure(classification, classItemCodes *string, orderItemCode *int64, parameters map[string]interface{}) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidPlan)
	}
	if p.Code <= 0 {
		return fmt.Errorf("%w: code must be positive", ErrInvalidPlan)
	}
	if p.IndependentDemands != IndependentDemandsNo && p.IndependentDemands != IndependentDemandsFromDate && p.IndependentDemands != IndependentDemandsAll {
		return fmt.Errorf("%w: independent_demands must be NO, FROM_DATE or ALL", ErrInvalidPlan)
	}
	types, err := normalizePlanningTypes(p.PlanningTypes)
	if err != nil {
		return err
	}
	p.PlanningTypes = types

	classification = normalizedOptionalString(classification)
	normalizedCodes, err := normalizeClassItemCodes(classItemCodes)
	if err != nil {
		return err
	}
	if normalizedCodes != nil && classification == nil {
		return fmt.Errorf("%w: classification is required when class_item_codes is informed", ErrInvalidPlan)
	}
	if orderItemCode != nil && *orderItemCode <= 0 {
		return fmt.Errorf("%w: order_item_code must be positive", ErrInvalidPlan)
	}
	if orderItemCode != nil && (classification != nil || normalizedCodes != nil) {
		return fmt.Errorf("%w: order_item_code cannot be combined with classification filters", ErrInvalidPlan)
	}

	if parameters == nil {
		parameters = map[string]interface{}{}
	}
	if p.IndependentDemands == IndependentDemandsFromDate {
		raw, ok := parameters["from_date"]
		value, stringOK := raw.(string)
		if !ok || !stringOK {
			return fmt.Errorf("%w: parameters.from_date is required for FROM_DATE", ErrInvalidPlan)
		}
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return fmt.Errorf("%w: parameters.from_date must use YYYY-MM-DD", ErrInvalidPlan)
		}
	}
	p.Classification, p.ClassItemCodes, p.OrderItemCode, p.Parameters = classification, normalizedCodes, orderItemCode, cloneParameters(parameters)
	return nil
}

func normalizePlanningTypes(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{"MRP"}, nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if _, ok := validPlanningTypes[value]; !ok {
			return nil, fmt.Errorf("%w: unsupported planning type %q", ErrInvalidPlan, value)
		}
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out, nil
}

func normalizeClassItemCodes(value *string) (*string, error) {
	value = normalizedOptionalString(value)
	if value == nil {
		return nil, nil
	}
	seen := map[string]struct{}{}
	codes := make([]string, 0)
	for _, part := range strings.Split(*value, ",") {
		code := strings.TrimSpace(part)
		if code == "" {
			return nil, fmt.Errorf("%w: class_item_codes must contain non-empty codes separated by commas", ErrInvalidPlan)
		}
		if _, ok := seen[code]; !ok {
			seen[code] = struct{}{}
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)
	normalized := strings.Join(codes, ",")
	return &normalized, nil
}

func normalizedOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	return &v
}
func cloneParameters(parameters map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(parameters))
	for key, value := range parameters {
		out[key] = value
	}
	return out
}
