package response

import (
	"time"

	"github.com/google/uuid"
)

// ItemUnitConversionResponse is the API representation of an item UOM conversion.
type ItemUnitConversionResponse struct {
	ID int64 `json:"id"`
	// ItemCode é o código de negócio do item (texto), como a tela o conhece.
	ItemCode string `json:"item_code"`
	ItemName string `json:"item_name,omitempty"`
	// LegacyCode é a chave numérica interna, mantida para integrações antigas.
	LegacyCode      int64     `json:"legacy_item_code"`
	Mask            string    `json:"mask,omitempty"`
	FromUOM         string    `json:"from_uom"`
	ToUOM           string    `json:"to_uom"`
	Factor          float64   `json:"factor"`
	RoundingPercent float64   `json:"rounding_percent"`
	ToleranceValue  float64   `json:"tolerance_value"`
	ToleranceType   string    `json:"tolerance_type"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	CreatedBy       uuid.UUID `json:"created_by"`
}
