package response

import (
	"time"

	"github.com/google/uuid"
)

// ItemUnitConversionResponse is the API representation of an item UOM conversion.
type ItemUnitConversionResponse struct {
	ID        int64     `json:"id"`
	ItemCode  int64     `json:"item_code"`
	FromUOM   string    `json:"from_uom"`
	ToUOM     string    `json:"to_uom"`
	Factor    float64   `json:"factor"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy uuid.UUID `json:"created_by"`
}
