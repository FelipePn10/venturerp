package valueobject

import "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"

type StructureNode struct {
	Component *entity.ItemStructure
	ItemCode  int64
	ItemDesc  string
	Level     int
	Mask      *string
	Children  []*StructureNode
}

func NewStructureNode(
	component *entity.ItemStructure,
	code int64,
	desc string,
	level int,
	mask *string,
) *StructureNode {
	return &StructureNode{
		Component: component,
		ItemCode:  code,
		ItemDesc:  desc,
		Level:     level,
		Mask:      mask,
		Children:  []*StructureNode{},
	}
}

func (n *StructureNode) AddChild(child *StructureNode) {
	n.Children = append(n.Children, child)
}
