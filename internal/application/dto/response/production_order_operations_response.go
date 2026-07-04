package response

import "time"

type ProductionOrderOperationResponse struct {
	ID                int64      `json:"id"`
	ProductionOrderID int64      `json:"production_order_id"`
	RouteOperationID  *int64     `json:"route_operation_id,omitempty"`
	Sequence          int        `json:"sequence"`
	OperationName     string     `json:"operation_name"`
	WorkCenterID      *int64     `json:"work_center_id,omitempty"`
	PlannedHours      float64    `json:"planned_hours"`
	SetupHours        float64    `json:"setup_hours"`
	ActualHours       float64    `json:"actual_hours"`
	Status            string     `json:"status"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	// ToolAlerts lists tools that reached their useful-life limit while completing
	// this operation (populated on DONE). Empty otherwise.
	ToolAlerts []string `json:"tool_alerts,omitempty"`
}
