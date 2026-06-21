package entity

import (
	"time"

	"github.com/shopspring/decimal"
)

type BomItems struct {
	ID            int64           `json:"id"`
	BomID         int64           `json:"bom_id"`
	ComponentID   int64           `json:"component_id"`
	Quantity      decimal.Decimal `json:"quantity"`
	Uom           string          `json:"uom"`
	ScrapPercent  decimal.Decimal `json:"scrap_percent"`
	OperationID   int64           `json:"operation_id"`
	CreatedAt     time.Time       `json:"created_at"`
	MaskComponent int64           `json:"mask_component"`
}
