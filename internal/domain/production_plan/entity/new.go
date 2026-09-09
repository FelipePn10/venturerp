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
	ErrInvalidPlan = errors.New("plano de produção inválido")
)

var validPlanningTypes = map[string]struct{}{
	"MRP": {}, "MIN_MAX": {}, "REORDER_POINT": {}, "MPS": {}, "KANBAN": {},
}

func NewProductionPlan(code int64, name, independentDemands string, groupSameDateOrders bool, planningTypes []string, createdBy uuid.UUID) (*ProductionPlan, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: informe o nome", ErrInvalidPlan)
	}
	if code <= 0 {
		return nil, fmt.Errorf("%w: o código deve ser maior que zero", ErrInvalidPlan)
	}
	switch independentDemands {
	case IndependentDemandsNo, IndependentDemandsFromDate, IndependentDemandsAll:
	default:
		return nil, fmt.Errorf("%w: as demandas independentes devem ser nenhuma, a partir de uma data ou todas", ErrInvalidPlan)
	}
	planningTypes, err := normalizePlanningTypes(planningTypes)
	if err != nil {
		return nil, err
	}
	if createdBy == uuid.Nil {
		return nil, fmt.Errorf("%w: é preciso um usuário autenticado para criar o registro", ErrInvalidPlan)
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
		return fmt.Errorf("%w: informe o nome", ErrInvalidPlan)
	}
	if p.Code <= 0 {
		return fmt.Errorf("%w: o código deve ser maior que zero", ErrInvalidPlan)
	}
	if p.IndependentDemands != IndependentDemandsNo && p.IndependentDemands != IndependentDemandsFromDate && p.IndependentDemands != IndependentDemandsAll {
		return fmt.Errorf("%w: as demandas independentes devem ser nenhuma, a partir de uma data ou todas", ErrInvalidPlan)
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
		return fmt.Errorf("%w: informe a classificação quando listar códigos de itens", ErrInvalidPlan)
	}
	if orderItemCode != nil && *orderItemCode <= 0 {
		return fmt.Errorf("%w: o item da ordem deve ser um código maior que zero", ErrInvalidPlan)
	}
	if orderItemCode != nil && (classification != nil || normalizedCodes != nil) {
		return fmt.Errorf("%w: o item da ordem não pode ser combinado com filtros de classificação", ErrInvalidPlan)
	}

	if parameters == nil {
		parameters = map[string]interface{}{}
	}
	if p.IndependentDemands == IndependentDemandsFromDate {
		raw, ok := parameters["from_date"]
		value, stringOK := raw.(string)
		if !ok || !stringOK {
			return fmt.Errorf("%w: informe a data inicial ao usar demandas a partir de uma data", ErrInvalidPlan)
		}
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return fmt.Errorf("%w: informe a data inicial no formato ano-mês-dia", ErrInvalidPlan)
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
			return nil, fmt.Errorf("%w: tipo de planejamento não suportado: %q", ErrInvalidPlan, value)
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
			return nil, fmt.Errorf("%w: informe os códigos separados por vírgula, sem itens vazios", ErrInvalidPlan)
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
