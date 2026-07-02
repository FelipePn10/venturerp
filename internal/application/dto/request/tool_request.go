package request

import "github.com/google/uuid"

type CreateToolDTO struct {
	Name      string    `json:"name"`
	ToolType  string    `json:"tool_type"`
	LifeType  string    `json:"life_type"`  // GOLPES | HORAS | PECAS
	LifeLimit float64   `json:"life_limit"` // 0 = no life tracking
	Cost      float64   `json:"cost"`
	CreatedBy uuid.UUID `json:"created_by"`
}

type UpdateToolDTO struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	ToolType  string  `json:"tool_type"`
	LifeType  string  `json:"life_type"`
	LifeLimit float64 `json:"life_limit"`
	Cost      float64 `json:"cost"`
	Status    string  `json:"status"` // ATIVA | MANUTENCAO | INATIVA
}

type AddRouteOpToolDTO struct {
	RouteOperationID int64   `json:"route_operation_id"`
	ToolID           int64   `json:"tool_id"`
	QtyRequired      float64 `json:"qty_required"`
}
