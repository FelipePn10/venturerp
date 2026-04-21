package structure_query

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/structure_query/service"
)

func MapNodes(nodes []*service.Node) []*response.StructureTreeNodeResponse {
	out := make([]*response.StructureTreeNodeResponse, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, mapNode(n))
	}
	return out
}

func mapNode(n *service.Node) *response.StructureTreeNodeResponse {
	children := make([]*response.StructureTreeNodeResponse, 0, len(n.Children))
	for _, c := range n.Children {
		children = append(children, mapNode(c))
	}

	return &response.StructureTreeNodeResponse{
		Component: response.StructureComponentResponse{
			ID:                n.Component.ID,
			ParentItemCode:    n.Component.ParentCode,
			ChildItemCode:     n.Component.ChildCode,
			ParentMask:        n.Component.ParentMask,
			Quantity:          n.Component.Quantity,
			UnitOfMeasurement: n.Component.UnitOfMeasurement,
			Health:            n.Component.Health,
			LossPercentage:    n.Component.LossPercentage,
			Position:          n.Component.Sequence,
			Notes:             n.Component.Notes,
			IsActive:          n.Component.IsActive,
			CreatedBy:         n.Component.CreatedBy,
			CreatedAt:         n.Component.CreatedAt,
			UpdatedAt:         n.Component.UpdatedAt,
		},
		EffectiveMask: n.EffectiveMask,
		RequiresMask:  n.RequiresMask,
		Level:         n.Level,
		Children:      children,
	}
}
