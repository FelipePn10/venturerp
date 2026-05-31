package request

import "github.com/google/uuid"

type CreateItemConversionDTO struct {
	ItemCode  int64     `json:"item_code"`
	FromUOM   string    `json:"from_uom"`
	ToUOM     string    `json:"to_uom"`
	Factor    float64   `json:"factor"` // 1 from_uom = factor × to_uom
	CreatedBy uuid.UUID `json:"created_by"`
}
