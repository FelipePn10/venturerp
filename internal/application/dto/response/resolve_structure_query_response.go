package response

type StructureQueryNode struct {
	ItemCode int64                 `json:"item_code"`
	Mask     *string               `json:"mask,omitempty"`
	Children []*StructureQueryNode `json:"children"`
}
