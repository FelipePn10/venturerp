package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CreateItemConversionDTO: o item chega pelo código de negócio (texto) — é assim
// que a tela de Conversões por Item (VSUP0130) o conhece.
type CreateItemConversionDTO struct {
	ItemCode        TextCode        `json:"item_code"`
	Mask            string          `json:"mask,omitempty"`
	FromUOM         string          `json:"from_uom"`
	ToUOM           string          `json:"to_uom"`
	Factor          decimal.Decimal `json:"factor"` // 1 from_uom = factor × to_uom
	RoundingPercent decimal.Decimal `json:"rounding_percent"`
	ToleranceValue  decimal.Decimal `json:"tolerance_value"`
	ToleranceType   string          `json:"tolerance_type"`
	CreatedBy       uuid.UUID       `json:"-"`
}
