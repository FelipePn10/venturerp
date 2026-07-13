package request

import "github.com/google/uuid"

type CreateItemConversionDTO struct {
	ItemCode        int64     `json:"item_code"`
	Mask            string    `json:"mask,omitempty"`
	FromUOM         string    `json:"from_uom"`
	ToUOM           string    `json:"to_uom"`
	Factor          float64   `json:"factor"` // 1 from_uom = factor × to_uom
	RoundingPercent float64   `json:"rounding_percent"`
	ToleranceValue  float64   `json:"tolerance_value"`
	ToleranceType   string    `json:"tolerance_type"`
	CreatedBy       uuid.UUID `json:"created_by"`
}
