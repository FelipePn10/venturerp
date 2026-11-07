package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateToolDTO struct {
	Name      string          `json:"name"`
	ToolType  string          `json:"tool_type"`
	LifeType  string          `json:"life_type"`  // GOLPES | HORAS | PECAS
	LifeLimit decimal.Decimal `json:"life_limit"` // 0 = no life tracking
	Cost      decimal.Decimal `json:"cost"`
	CreatedBy uuid.UUID       `json:"created_by"`
	// ItemCode liga a ficha ao item marcado como ferramenta.
	ItemCode *int64 `json:"item_code,omitempty"`
}

type UpdateToolDTO struct {
	ID        int64           `json:"id"`
	Name      string          `json:"name"`
	ToolType  string          `json:"tool_type"`
	LifeType  string          `json:"life_type"`
	LifeLimit decimal.Decimal `json:"life_limit"`
	Cost      decimal.Decimal `json:"cost"`
	Status    string          `json:"status"` // ATIVA | MANUTENCAO | INATIVA
	// ItemCode ausente preserva o vínculo; null desfaz.
	ItemCode Opcional[int64] `json:"item_code"`
}

type AddRouteOpToolDTO struct {
	RouteOperationID int64           `json:"route_operation_id"`
	ToolID           int64           `json:"tool_id"`
	QtyRequired      decimal.Decimal `json:"qty_required"`
}
