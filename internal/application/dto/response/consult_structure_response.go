package response

import "time"

type WhereUsedRowResponse struct {
	Level             int     `json:"level"`
	ParentCode        int64   `json:"parent_code"`
	ParentDescription string  `json:"parent_description"`
	ChildCode         int64   `json:"child_code"`
	Quantity          float64 `json:"quantity"`
	LossPercentage    float64 `json:"loss_percentage"`
	Sequence          int     `json:"sequence"`
	ParentMask        *string `json:"parent_mask,omitempty"`
}

type WhereUsedResponse struct {
	ItemCode int64                  `json:"item_code"`
	Rows     []WhereUsedRowResponse `json:"rows"`
}

// ConsultStructureRowResponse representa uma linha da grade VENG0401.
type ConsultStructureRowResponse struct {
	Level             int        `json:"level"`
	ParentCode        int64      `json:"parent_code"`
	ItemCode          int64      `json:"item_code"`
	Description       string     `json:"description"`
	Sequence          int        `json:"sequence"`
	StartDate         *time.Time `json:"start_date,omitempty"`
	EndDate           *time.Time `json:"end_date,omitempty"`
	Quantity          float64    `json:"quantity"`
	WarehouseCode     int64      `json:"warehouse_code"`
	LossFormula       *string    `json:"loss_formula,omitempty"`
	LossPercentage    float64    `json:"loss_percentage"`
	CorrectedQuantity float64    `json:"corrected_quantity"`
	StructureType     int16      `json:"structure_type"`
	Mask              *string    `json:"mask,omitempty"`
}

// ConsultStructureResponse é o payload completo retornado pelo endpoint VENG0401.
type ConsultStructureResponse struct {
	RootItemCode int64                         `json:"root_item_code"`
	Mask         string                        `json:"mask,omitempty"`
	Rows         []ConsultStructureRowResponse `json:"rows"`
}
