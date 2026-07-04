package mapper

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/structure/valueobject"
)

// domínio → response
// MapNodeToResponse converte um StructureNode do domínio para DTO de resposta.
func MapNodeToResponse(node *valueobject.StructureNode) *response.StructureTreeNodeResponse {
	if node == nil || node.Component == nil {
		return nil
	}

	comp := node.Component

	componentResp := response.StructureComponentResponse{
		ID:                 comp.ID,
		ParentCode:         node.ItemCode,
		ChildCode:          node.ItemCode,
		ChildDescription:   node.ItemDesc,
		ParentMask:         comp.ParentMask,
		IsGeneric:          comp.IsGeneric(),
		Quantity:           comp.Quantity,
		EffectiveQuantity:  comp.EffectiveQuantity(),
		UnitOfMeasurement:  comp.UnitOfMeasurement,
		Health:             comp.Health,
		LossPercentage:     comp.LossPercentage,
		LossFormula:        comp.LossFormula,
		Sequence:           comp.Sequence,
		Notes:              comp.Notes,
		StartDate:          comp.StartDate,
		EndDate:            comp.EndDate,
		IsCoproduct:        comp.IsCoproduct,
		IsFixedQty:         comp.IsFixedQty,
		SubstituteGroup:    comp.SubstituteGroup,
		SubstitutePriority: comp.SubstitutePriority,
		IsActive:           comp.IsActive,
		CreatedBy:          comp.CreatedBy,
		CreatedAt:          comp.CreatedAt,
		UpdatedAt:          comp.UpdatedAt,
	}

	resp := &response.StructureTreeNodeResponse{
		Component: componentResp,
		Mask:      node.Mask, // atualizado (antes era ResolvedMask)
		Level:     node.Level,
		Children:  make([]*response.StructureTreeNodeResponse, 0, len(node.Children)),
	}

	for _, child := range node.Children {
		mapped := MapNodeToResponse(child)
		if mapped != nil {
			resp.Children = append(resp.Children, mapped)
		}
	}

	return resp
}

// MapNodes converte lista de nós
func MapNodes(nodes []*valueobject.StructureNode) []*response.StructureTreeNodeResponse {
	result := make([]*response.StructureTreeNodeResponse, 0, len(nodes))

	for _, n := range nodes {
		if mapped := MapNodeToResponse(n); mapped != nil {
			result = append(result, mapped)
		}
	}

	return result
}

// CountNodes conta recursivamente o total de nós
func CountNodes(nodes []*response.StructureTreeNodeResponse) int {
	total := 0
	for _, n := range nodes {
		total++
		total += CountNodes(n.Children)
	}
	return total
}

// MaxLevel retorna profundidade máxima
func MaxLevel(nodes []*response.StructureTreeNodeResponse) int {
	max := 0

	for _, n := range nodes {
		if n.Level > max {
			max = n.Level
		}

		childMax := MaxLevel(n.Children)
		if childMax > max {
			max = childMax
		}
	}

	return max
}
