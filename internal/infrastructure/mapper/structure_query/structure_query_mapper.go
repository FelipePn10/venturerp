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

	// Quantity já vem resolvida pelo resolver (fórmula avaliada quando possível);
	// a quantidade fixa cadastrada continua visível em NominalQuantity.
	quantity := n.Quantity
	if quantity == 0 && !n.FormulaApplied {
		quantity = n.Component.Quantity
	}
	return &response.StructureTreeNodeResponse{
		Component: response.StructureComponentResponse{
			ID:                 n.Component.ID,
			ParentCode:         n.Component.ParentCode,
			ChildCode:          n.Component.ChildCode,
			ChildDescription:   n.Component.ChildDescription,
			ParentMask:         n.Component.ParentMask,
			Quantity:           quantity,
			NominalQuantity:    n.Component.Quantity,
			QuantityFormula:    n.Component.QuantityFormula,
			QuantityRounding:   n.Component.QuantityRounding,
			QuantityScale:      n.Component.QuantityScale,
			FormulaApplied:     n.FormulaApplied,
			FormulaVariables:   n.Component.FormulaVariables(),
			EffectiveQuantity:  quantity * (1 + n.Component.LossPercentage/100),
			UnitOfMeasurement:  n.Component.UnitOfMeasurement,
			Health:             n.Component.Health,
			LossPercentage:     n.Component.LossPercentage,
			LossFormula:        n.Component.LossFormula,
			Sequence:           n.Component.Sequence,
			Notes:              n.Component.Notes,
			StartDate:          n.Component.StartDate,
			EndDate:            n.Component.EndDate,
			IsCoproduct:        n.Component.IsCoproduct,
			IsFixedQty:         n.Component.IsFixedQty,
			SubstituteGroup:    n.Component.SubstituteGroup,
			SubstitutePriority: n.Component.SubstitutePriority,
			Inherit:             n.Component.Inherit,
			WarehouseCode:       n.Component.WarehouseCode,
			LineWarehouseCode:   n.Component.LineWarehouseCode,
			SetupLoss:           n.Component.SetupLoss,
			CostLossType:        n.Component.CostLossType,
			CostLoss:            n.Component.CostLoss,
			CostCenterCode:      n.Component.CostCenterCode,
			IsCriticalMPS:       n.Component.IsCriticalMPS,
			GeneratesInspection: n.Component.GeneratesInspection,
			IsActive:           n.Component.IsActive,
			CreatedBy:          n.Component.CreatedBy,
			CreatedAt:          n.Component.CreatedAt,
			UpdatedAt:          n.Component.UpdatedAt,
		},
		EffectiveMask: n.EffectiveMask,
		RequiresMask:  n.RequiresMask,
		Level:         n.Level,
		Children:      children,
	}
}
