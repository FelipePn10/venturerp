package response

import "time"

type ToolResponse struct {
	ID               int64     `json:"id"`
	Code             int64     `json:"code"`
	Name             string    `json:"name"`
	ToolType         string    `json:"tool_type"`
	LifeType         string    `json:"life_type"`
	LifeLimit        float64   `json:"life_limit"`
	LifeUsed         float64   `json:"life_used"`
	RemainingLife    float64   `json:"remaining_life"` // -1 when untracked
	NeedsReplacement bool      `json:"needs_replacement"`
	Cost             float64   `json:"cost"`
	Status           string    `json:"status"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
}

type RouteOpToolResponse struct {
	ID               int64   `json:"id"`
	RouteOperationID int64   `json:"route_operation_id"`
	ToolID           int64   `json:"tool_id"`
	ToolCode         int64   `json:"tool_code"`
	ToolName         string  `json:"tool_name"`
	QtyRequired      float64 `json:"qty_required"`
	LifeType         string  `json:"life_type,omitempty"`
	LifeLimit        float64 `json:"life_limit"`
	LifeUsed         float64 `json:"life_used"`
	NeedsReplacement bool    `json:"needs_replacement"`
	Status           string  `json:"status,omitempty"`
}
